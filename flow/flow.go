package flow

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"syscall"
)

const (
	// Window modes
	Unbuffered = -1      // Aggressive flushing
	Buffered   = 0       // No flushing until EOF
	Windowed   = 1048576 // 1 MiB max buffer length
)

// RenderFunc is a function that renders markdown to output bytes
type RenderFunc func(data []byte) ([]byte, error)

// Config configures the streaming buffer behavior
type Config struct {
	Window int64      // -1: minimal buffering (aggressive flush), 0: maximum buffering (no flush until EOF), N: bounded at N bytes
	Render RenderFunc // glamour renderer (byte-based)
}

// Buffer handles incremental markdown rendering with configurable buffering
type Buffer struct {
	config      Config
	writer      io.Writer // output writer
	accum       []byte    // accumulated markdown
	written     int64     // total bytes written
	inCodeFence bool      // currently inside ``` code fence
	hadSplits   bool      // track if we actually split at buffer boundaries
}

// NewBuffer creates a new streaming buffer
func NewBuffer(config Config, w io.Writer) *Buffer {
	return &Buffer{
		config:      config,
		writer:      w,
		accum:       make([]byte, 0, 4096), // 4 KiB standard buffer
		inCodeFence: false,
	}
}

// calculateFenceState determines if we're inside a code fence for given data
// When resetFirst is true, it clears state before calculation (used after splits)
func (b *Buffer) calculateFenceState(data []byte, resetFirst bool) {
	if resetFirst {
		b.inCodeFence = false
	}

	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		// Check for code fence markers (``` or ~~~)
		if bytes.HasPrefix(trimmed, []byte("```")) || bytes.HasPrefix(trimmed, []byte("~~~")) {
			// Toggle fence state for standard triple-backtick fences
			b.inCodeFence = !b.inCodeFence
		}
	}
}

// Process reads markdown from r and streams rendered output via Output callback
func (b *Buffer) Process(ctx context.Context, r io.Reader) error {
	if ctx.Err() != nil {
		// Distinguish between different cancellation types:
		// - context.Canceled: clean user cancellation → return nil
		// - context.DeadlineExceeded: timeout → return error for proper exit code
		if ctx.Err() == context.DeadlineExceeded {
			return ctx.Err() // Timeout needs error for exit code
		}
		if ctx.Err() == context.Canceled {
			return nil // Clean cancellation
		}
		return ctx.Err()
	}

	// Create cancellable context for this process
	processCtx, processCancel := context.WithCancel(ctx)
	defer processCancel() // Ensure goroutine cleanup

	// Channel for read results
	type readResult struct {
		data []byte
		err  error
	}
	// Buffer the channel to avoid blocking on send
	readChan := make(chan readResult, 1)
	prebuffer := make([]byte, 0, 4096)
	frontmatterRemoved := false
	currentLineLen := 0
	maxPrebuffer := int64(Windowed)
	if b.config.Window > Buffered && b.config.Window < maxPrebuffer {
		maxPrebuffer = b.config.Window
	}

	// Start goroutine to read from input
	go func() {
		defer close(readChan)
		// Simple read buffer
		buf := make([]byte, 4096) // 4 KiB standard buffer
		for {
			// Check for context cancellation before reading
			select {
			case <-processCtx.Done():
				return
			default:
			}

			n, err := r.Read(buf)
			if n > 0 {
				// Send copy of data to avoid race conditions
				data := make([]byte, n)
				copy(data, buf[:n])

				// Try to send with context check
				select {
				case <-processCtx.Done():
					return
				case readChan <- readResult{data: data, err: err}:
				}
			}

			// If we have an error (including EOF), send it
			// Note: n > 0 with err != nil is valid (e.g., last read before EOF)
			if err != nil {
				// If we already sent data with the error above, don't send again
				if n == 0 {
					select {
					case <-processCtx.Done():
						return
					case readChan <- readResult{err: err}:
					}
				}
				return // Exit loop on any error including EOF
			}
		}
	}()

	// Main processing loop
	for {
		select {
		case <-processCtx.Done():
			// Context cancelled - flush accumulated content
			// Tests expect partial output to be rendered on interrupt
			// Use flush(false) to avoid glamour's EOF double-newline
			if len(b.accum) > 0 {
				return b.flush()
			}
			return nil

		case result, ok := <-readChan:
			if !ok {
				// Channel closed unexpectedly
				if len(b.accum) > 0 {
					return b.flush()
				}
				return nil
			}

			// Process the data
			if len(result.data) > 0 {

				// Handle frontmatter detection ONLY at start of document
				if !frontmatterRemoved && b.written == 0 {
					prebuffer = append(prebuffer, result.data...)

					// Immediately check first byte to avoid latency
					if len(prebuffer) > 0 && prebuffer[0] != '-' {
						// First byte is not '-', can't be frontmatter
						frontmatterRemoved = true
						b.calculateFenceState(prebuffer, false)
						b.accum = append(b.accum, prebuffer...)
						prebuffer = nil
					} else if len(prebuffer) >= 3 {
						// Check for frontmatter start (need at least "---")
						if !bytes.HasPrefix(prebuffer, []byte("---")) {
							// Doesn't start with "---", not frontmatter
							frontmatterRemoved = true
							b.calculateFenceState(prebuffer, false)
							b.accum = append(b.accum, prebuffer...)
							prebuffer = nil
						} else if len(prebuffer) >= 4 {
							// Have "---" and 4th byte, check for newline
							if prebuffer[3] == '\n' || (len(prebuffer) >= 5 && prebuffer[3] == '\r' && prebuffer[4] == '\n') {
								// This is frontmatter, look for closing --- on a line
								// Simple inline search replaces 41-line function
								searchStart := 4 // Skip opening "---\n"
								if idx := bytes.Index(prebuffer[searchStart:], []byte("\n---\n")); idx >= 0 {
									// Found closing frontmatter
									endIdx := searchStart + idx + 5 // Skip past "\n---\n"
									prebuffer = prebuffer[endIdx:]
									frontmatterRemoved = true
									// Add remaining prebuffer to accumulator
									if len(prebuffer) > 0 {
										b.calculateFenceState(prebuffer, false)
										b.accum = append(b.accum, prebuffer...)
										prebuffer = nil
									}
								} else if idx := bytes.Index(prebuffer[searchStart:], []byte("\n---")); idx >= 0 &&
									searchStart+idx+4 == len(prebuffer) {
									// Frontmatter ends at EOF
									frontmatterRemoved = true
									prebuffer = nil
								} else if int64(len(prebuffer)) >= maxPrebuffer {
									// Reached max prebuffer size without finding end
									// Assume no valid frontmatter or malformed
									frontmatterRemoved = true
									b.calculateFenceState(prebuffer, false)
									b.accum = append(b.accum, prebuffer...)
									prebuffer = nil
								}
								// Otherwise keep buffering for frontmatter end
							} else {
								// "---" not followed by newline, not frontmatter
								frontmatterRemoved = true
								b.calculateFenceState(prebuffer, false)
								b.accum = append(b.accum, prebuffer...)
								prebuffer = nil
							}
						}
						// Otherwise wait for 4th byte to check newline
					}
					// Otherwise keep buffering until we have enough bytes
				} else {
					// Normal accumulation after frontmatter handled
					b.calculateFenceState(result.data, false)
					b.accum = append(b.accum, result.data...)

					// Track line length to prevent glamour hanging on pathological input
					if idx := bytes.LastIndexByte(result.data, '\n'); idx >= 0 {
						// Found newline, reset line length counter
						currentLineLen = len(result.data) - idx - 1
					} else {
						// No newline, accumulate line length
						currentLineLen += len(result.data)
					}

					// Inject newline if line exceeds limit to prevent glamour hanging
					if currentLineLen > Windowed {
						b.accum = append(b.accum, '\n')
						currentLineLen = 0
					}
				}

				// Check if we should flush based on buffer size
				bufferSize := int64(len(b.accum))

				// Force flush if we exceed hard limit
				// This is a safety net to prevent OOM from unbounded input
				// In normal operation, shouldFlush() and flushToSafeBoundary() handle this
				if bufferSize > Windowed {
					b.flushToSafeBoundary(true)
				} else if b.shouldFlush() {
					// Force flush if we've accumulated way too much
					force := b.config.Window > Buffered && bufferSize > b.config.Window*2
					if err := b.flushToSafeBoundary(force); err != nil {
						return fmt.Errorf("opportunistic flush failed: %w", err)
					}
				}

				// Hard limit enforcement - split pathological input at safe boundaries
				if len(b.accum) >= Windowed {
					// Split at safe chunk boundary to avoid glamour limits
					if err := b.splitAndFlush(); err != nil {
						return fmt.Errorf("pathological split failed: %w", err)
					}
				}
			}

			// Handle errors
			if result.err == io.EOF {

				// Handle any remaining prebuffer
				if !frontmatterRemoved && len(prebuffer) > 0 {
					// EOF reached while detecting frontmatter
					// Treat remaining prebuffer as content
					b.calculateFenceState(prebuffer, false)
					b.accum = append(b.accum, prebuffer...)
				}

				// Final flush of all remaining content
				if len(b.accum) > 0 {
					return b.flush()
				}
				// Empty input special case - glamour outputs \n\n for empty input
				if b.written == 0 {
					// Never had any output, render empty string to match glamour
					return b.render([]byte{})
				}
				return nil
			}

			if result.err != nil {
				return result.err
			}
		}
	}
}

// shouldFlush determines if buffer should be flushed based on flow config
func (b *Buffer) shouldFlush() bool {
	if b.config.Window == Buffered {
		// No flow: never flush until EOF
		return false
	}
	if b.config.Window < Buffered {
		// Aggressive flow: flush at ANY newline
		return bytes.Contains(b.accum, []byte("\n"))
	}
	// Bounded flow: flush when exceeding size threshold
	return int64(len(b.accum)) >= b.config.Window
}

// findSafeBoundary finds the last safe split point
func (b *Buffer) findSafeBoundary() int {
	// Never split inside code fence to preserve syntax highlighting integrity
	if b.inCodeFence {
		return -1
	}

	// Priority 1: Complete code blocks
	if idx := b.findCodeBlockBoundary(b.accum); idx > 0 {
		return idx
	}

	// Priority 2: Double newlines (paragraph boundaries)
	if idx := bytes.LastIndex(b.accum, []byte("\n\n")); idx >= 0 {
		return idx + 2
	}

	// Priority 3: Any newline as last resort
	if idx := bytes.LastIndex(b.accum, []byte("\n")); idx >= 0 {
		return idx + 1
	}

	// No boundary found
	return -1
}

// findCodeBlockBoundary finds the end of a complete code block if present
func (b *Buffer) findCodeBlockBoundary(data []byte) int {
	// Simple open/close tracking (markdown doesn't support nested fences)
	inFence := false
	lastCompleteEnd := -1
	idx := 0

	for idx < len(data) {
		// Check if we're at start of line
		if idx == 0 || data[idx-1] == '\n' {
			// Check for fence marker (``` or ~~~)
			if idx+2 < len(data) && ((data[idx] == '`' && data[idx+1] == '`' && data[idx+2] == '`') ||
				(data[idx] == '~' && data[idx+1] == '~' && data[idx+2] == '~')) {

				// Toggle fence state
				inFence = !inFence

				// Skip to end of line
				for idx < len(data) && data[idx] != '\n' {
					idx++
				}
				if idx < len(data) {
					idx++ // Include newline
				}

				// If we just closed a fence, mark this position
				if !inFence {
					lastCompleteEnd = idx
				}
				continue
			}
		}
		idx++
	}

	return lastCompleteEnd
}

// flushToSafeBoundary flushes content up to the last safe boundary
func (b *Buffer) flushToSafeBoundary(force bool) error {
	if len(b.accum) == 0 {
		return nil
	}

	// For aggressive flow (flow=-1), flush at ANY newline boundary
	if b.config.Window < Buffered {
		// Find the last newline in the buffer
		if idx := bytes.LastIndex(b.accum, []byte("\n")); idx >= 0 {
			boundary := idx + 1
			toRender := b.accum[:boundary]
			b.accum = b.accum[boundary:]
			// Only mark as split if there's remaining content
			if len(b.accum) > 0 {
				b.hadSplits = true
			}
			err := b.render(toRender)
			if err != nil {
				return fmt.Errorf("render failed after %d bytes: %w", b.written, err)
			}
			return nil
		}
		// No newline found, check for pathological input
		if force || len(b.accum) >= Windowed {
			// Split at safe boundary to avoid glamour hanging
			return b.splitAndFlush()
		}
		return nil
	}

	boundary := b.findSafeBoundary()
	if boundary <= 0 {
		if force {
			// Forced flush: render everything
			// CRITICAL: Never signal EOF for intermediate chunks!
			return b.flush()
		}
		// If we've accumulated significantly over threshold, flush at line boundary
		// Use a simple 2x threshold - if we're over 2x the window size, force a flush
		if b.config.Window > Buffered && int64(len(b.accum)) > b.config.Window*2 {
			// Find a line boundary as fallback
			if idx := bytes.LastIndex(b.accum, []byte("\n")); idx > 0 {
				boundary = idx + 1
			} else {
				// No line boundary either, split at safe chunk size
				return b.splitAndFlush()
			}
		} else {
			// No safe boundary: keep accumulating
			return nil
		}
	}

	// Split at boundary
	toRender := b.accum[:boundary]
	b.accum = b.accum[boundary:]
	// Only mark as split if there's remaining content (not just flushing everything)
	if len(b.accum) > 0 {
		b.hadSplits = true
	}

	// Recalculate fence state for remaining content after split
	b.calculateFenceState(b.accum, true)

	// Debug output

	// Reset cache since buffer changed

	// This is an intermediate flush at a boundary, never final
	// Only explicit flush(true) calls should be final
	err := b.render(toRender)
	if err != nil {
		return fmt.Errorf("split flush failed: %w", err)
	}
	return nil
}

// flush renders all accumulated content
func (b *Buffer) flush() error {
	if len(b.accum) == 0 {
		return nil
	}

	toRender := b.accum
	b.accum = b.accum[:0] // Clear but keep capacity

	// Reset cache since buffer is cleared

	// Reset fence state after flush (accumulator is now empty)
	b.inCodeFence = false

	return b.render(toRender)
}

// splitAndFlush splits pathological input at safe boundaries to avoid glamour hanging
func (b *Buffer) splitAndFlush() error {
	if len(b.accum) == 0 {
		return nil
	}

	// Split at SafeChunkSize boundary to stay under glamour's limit
	for len(b.accum) >= Windowed {
		// Extract a safe chunk
		chunk := b.accum[:Windowed]
		b.accum = b.accum[Windowed:]
		b.hadSplits = true // Mark that we've split the input

		// Recalculate fence state for remaining content after split
		b.calculateFenceState(b.accum, true)

		// Render the chunk
		if err := b.render(chunk); err != nil {
			return fmt.Errorf("flush render failed: %w", err)
		}
	}

	// If there's remaining data less than chunk size, flush it
	if len(b.accum) > 0 {
		return b.flush()
	}

	return nil
}

// normalizeSpacing ensures consistent spacing after code blocks, headers, and HR
func (b *Buffer) normalizeSpacing(output []byte) []byte {
	lines := bytes.Split(output, []byte("\n"))
	result := make([]byte, 0, len(output)+100) // Pre-allocate extra for spacing additions
	inCode := false
	wasHeader := false
	wasHR := false

	for i, line := range lines {
		if i > 0 {
			result = append(result, '\n')
		}

		// Check if line is code (starts with 4 spaces after the 2-space margin)
		isCode := len(line) >= 4 && bytes.HasPrefix(line, []byte("    "))

		// Check if line is a header (starts with "  #" after glamour's 2-space margin)
		isHeader := len(line) >= 3 && bytes.HasPrefix(line, []byte("  #"))

		// Check if line is a horizontal rule (glamour renders as "  --------")
		isHR := bytes.Equal(bytes.TrimSpace(line), []byte("--------"))

		// Handle headers and HRs - ensure empty line after them has two spaces
		if (wasHeader || wasHR) && len(line) == 0 {
			// After a header or HR, empty lines should have two spaces
			line = []byte("  ")
		}
		wasHeader = isHeader
		wasHR = isHR

		// Track code block state transitions
		if !inCode && isCode {
			inCode = true
		} else if inCode && !isCode {
			// Just exited code block - ensure proper spacing
			inCode = false
			if len(line) == 0 {
				// Replace empty line with two-space line
				line = []byte("  ")
			} else if !bytes.Equal(line, []byte("  ")) && len(bytes.TrimSpace(line)) > 0 {
				// Content immediately after code - add spacing line
				result = append(result, []byte("  \n")...)
			}
		}

		result = append(result, line...)
	}

	return result
}

// render processes markdown through glamour and outputs directly to writer
func (b *Buffer) render(data []byte) error {
	output, err := b.config.Render(data)
	if err != nil {
		return err
	}

	// Minimal spacing normalization for code blocks
	if b.config.Window > Buffered && b.hadSplits {
		output = b.normalizeSpacing(output)
	}

	// Strip leading newline from non-first chunks
	if b.written > 0 && len(output) > 0 && output[0] == '\n' {
		output = output[1:]
	}

	// Skip completely empty output
	if len(output) == 0 {
		return nil
	}

	if _, err := b.writer.Write(output); err != nil {
		// Handle EPIPE by exiting immediately - this is normal Unix pipeline behavior
		if errors.Is(err, syscall.EPIPE) {
			return nil // Exit cleanly when downstream closes - this terminates the process
		}
		return fmt.Errorf("write failed after %d bytes: %w", b.written, err)
	}

	b.written += int64(len(output))
	return nil
}

// Flow is the main entry point for streaming markdown rendering
func Flow(ctx context.Context, r io.Reader, w io.Writer, window int64, render RenderFunc) error {
	// Validate inputs
	switch {
	case ctx == nil:
		return fmt.Errorf("context cannot be nil")
	case r == nil:
		return fmt.Errorf("input reader cannot be nil")
	case w == nil:
		return fmt.Errorf("output writer cannot be nil")
	case render == nil:
		return fmt.Errorf("render function cannot be nil")
	case window < Unbuffered || (window > 0 && window > Windowed):
		return fmt.Errorf("window must be -1 (block), 0 (stream), or 1-%d (bytes)", Windowed)
	}

	// Create buffer configuration
	config := Config{
		Window: window,
		Render: render,
	}

	// Process the stream
	return NewBuffer(config, w).Process(ctx, r)
}
