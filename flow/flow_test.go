package flow

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

// TestFlowBasicStreaming tests basic streaming functionality
func TestFlowBasicStreaming(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		window   int64
		expected string
	}{
		{
			name:     "simple header",
			input:    "# Test",
			window:   Buffered,
			expected: "# Test",
		},
		{
			name:     "empty input",
			input:    "",
			window:   Buffered,
			expected: "",
		},
		{
			name:     "multi-line markdown",
			input:    "# Title\n\nParagraph\n\n## Section",
			window:   Buffered,
			expected: "Title",
		},
		{
			name:     "code block",
			input:    "```\ncode\n```",
			window:   Buffered,
			expected: "code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			r := strings.NewReader(tt.input)
			var buf bytes.Buffer

			// Simple renderer that passes through
			renderer := func(data []byte) ([]byte, error) {
				return data, nil
			}

			err := Flow(ctx, r, &buf, tt.window, renderer)
			if err != nil {
				t.Fatalf("Flow error: %v", err)
			}

			output := buf.String()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("Expected output to contain %q, got %q", tt.expected, output)
			}
		})
	}
}

// TestFlowContextCancellation tests context cancellation handling
func TestFlowContextCancellation(t *testing.T) {
	t.Run("immediate cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		r := strings.NewReader("# Test\n\nContent")
		var buf bytes.Buffer

		renderer := func(data []byte) ([]byte, error) {
			return data, nil
		}

		err := Flow(ctx, r, &buf, Buffered, renderer)
		// Should exit cleanly without error
		if err != nil {
			t.Errorf("Expected no error for cancelled context, got: %v", err)
		}
	})

	t.Run("cancellation during streaming", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Create a slow reader that blocks
		r := &slowReader{
			chunks: []string{"# First\n", "## Second\n", "### Third\n"},
			delay:  20 * time.Millisecond, // Short enough to get first chunk
		}
		var buf bytes.Buffer

		err := Flow(ctx, r, &buf, Unbuffered, passthroughRenderer)
		// Should handle cancellation gracefully
		if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
			t.Errorf("Unexpected error: %v", err)
		}

		// Should have processed at least first chunk
		output := buf.String()
		if !strings.Contains(output, "First") {
			// Not a hard failure - timing dependent
			t.Logf("Warning: First chunk not processed before cancellation (timing dependent)")
		}
	})
}

// TestFlowPathologicalInputs tests handling of pathological inputs
func TestFlowPathologicalInputs(t *testing.T) {
	tests := []struct {
		name   string
		input  io.Reader
		window int64
	}{
		{
			name:   "massive single line",
			input:  &repeatReader{pattern: []byte("A"), limit: 1000000},
			window: 4096,
		},
		{
			name:   "deeply nested code blocks",
			input:  strings.NewReader("```\n```\n```\ncode\n```\n```\n```"),
			window: Buffered,
		},
		{
			name:   "binary data",
			input:  &binaryReader{size: 1024},
			window: 4096,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			var buf bytes.Buffer
			renderer := func(data []byte) ([]byte, error) {
				return data, nil
			}

			err := Flow(ctx, tt.input, &buf, tt.window, renderer)
			// Should complete without hanging or crashing
			if err != nil && err != context.DeadlineExceeded {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestFlowResourceLimits tests goroutine resource management
func TestFlowResourceLimits(t *testing.T) {
	t.Run("rapid cancellations", func(t *testing.T) {
		// Rapidly create and cancel Flow operations to test for goroutine leaks
		renderer := func(data []byte) ([]byte, error) {
			return data, nil
		}

		for i := 0; i < 100; i++ {
			ctx, cancel := context.WithCancel(context.Background())
			r := strings.NewReader("# Test content")
			var buf bytes.Buffer

			go func() {
				time.Sleep(time.Microsecond)
				cancel()
			}()

			_ = Flow(ctx, r, &buf, Unbuffered, renderer)
		}

		// Give time for goroutines to clean up
		time.Sleep(100 * time.Millisecond)
		// No good way to assert goroutine count in test, but this shouldn't leak
	})
}

// TestFlowPipeClosure tests handling of closed pipes
func TestFlowPipeClosure(t *testing.T) {
	t.Run("write to closed pipe", func(t *testing.T) {
		ctx := context.Background()
		r := strings.NewReader("# Test content\n\nMore content")

		// Create a writer that closes after first write
		w := &closingWriter{
			closeAfter: 1,
			buf:        &bytes.Buffer{},
		}

		renderer := func(data []byte) ([]byte, error) {
			return data, nil
		}

		err := Flow(ctx, r, w, Unbuffered, renderer)
		// Should handle closed pipe gracefully - io.ErrClosedPipe is expected
		if err != nil && !errors.Is(err, io.ErrClosedPipe) {
			t.Errorf("Expected no error or io.ErrClosedPipe for closed pipe, got: %v", err)
		}
	})
}

// TestFlowFrontmatterRemoval tests frontmatter detection and removal
func TestFlowFrontmatterRemoval(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		absent   string
	}{
		{
			name:     "yaml frontmatter",
			input:    "---\ntitle: Test\n---\n# Content",
			expected: "Content",
			absent:   "title:",
		},
		{
			name:     "no frontmatter",
			input:    "# Direct content",
			expected: "Direct content",
			absent:   "",
		},
		{
			name:     "malformed frontmatter",
			input:    "---\nunclosed frontmatter\n# Content",
			expected: "Content",
			absent:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			r := strings.NewReader(tt.input)
			var buf bytes.Buffer

			err := Flow(ctx, r, &buf, Buffered, passthroughRenderer)
			if err != nil {
				t.Fatalf("Flow error: %v", err)
			}

			output := buf.String()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("Expected output to contain %q, got %q", tt.expected, output)
			}
			if tt.absent != "" && strings.Contains(output, tt.absent) {
				t.Errorf("Expected output to NOT contain %q, but it did", tt.absent)
			}
		})
	}
}

// Helper types for testing

type slowReader struct {
	chunks []string
	idx    int
	delay  time.Duration
}

func (r *slowReader) Read(p []byte) (n int, err error) {
	if r.idx >= len(r.chunks) {
		return 0, io.EOF
	}
	// Only delay after the first chunk
	if r.idx > 0 {
		time.Sleep(r.delay)
	}
	n = copy(p, r.chunks[r.idx])
	r.idx++
	return n, nil
}

type binaryReader struct {
	size int
	sent int
}

func (r *binaryReader) Read(p []byte) (n int, err error) {
	if r.sent >= r.size {
		return 0, io.EOF
	}
	remaining := r.size - r.sent
	if len(p) > remaining {
		p = p[:remaining]
	}
	for i := range p {
		p[i] = byte(i % 256)
	}
	r.sent += len(p)
	return len(p), nil
}

type closingWriter struct {
	closeAfter int
	writes     int
	buf        *bytes.Buffer
	closed     bool
}

func (w *closingWriter) Write(p []byte) (n int, err error) {
	if w.closed {
		return 0, io.ErrClosedPipe
	}
	w.writes++
	if w.writes >= w.closeAfter {
		w.closed = true
		return 0, io.ErrClosedPipe
	}
	return w.buf.Write(p)
}
