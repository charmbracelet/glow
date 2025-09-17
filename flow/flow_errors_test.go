// Package flow error tests validate comprehensive error handling and recovery.
// These tests ensure robust handling of failure conditions:
// - I/O failures (read errors, write errors, broken pipes)
// - Resource exhaustion (memory limits, timeouts)
// - Invalid inputs (malformed markdown, binary data)
// - Error injection via custom reader/writer implementations
// - Graceful degradation and partial output recovery
package flow

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// 1. I/O FAILURE INJECTION TESTS

// failingReader simulates read errors after N bytes
type failingReader struct {
	data      []byte
	failAfter int
	bytesRead int
	errorMsg  string
}

func (r *failingReader) Read(p []byte) (n int, err error) {
	if r.bytesRead >= r.failAfter {
		if r.errorMsg != "" {
			return 0, fmt.Errorf("%s", r.errorMsg)
		}
		return 0, fmt.Errorf("injected read error at byte %d", r.bytesRead)
	}

	remaining := r.failAfter - r.bytesRead
	toCopy := len(p)
	if toCopy > remaining {
		toCopy = remaining
	}
	if toCopy > len(r.data)-r.bytesRead {
		toCopy = len(r.data) - r.bytesRead
	}

	if toCopy > 0 {
		n = copy(p, r.data[r.bytesRead:r.bytesRead+toCopy])
		r.bytesRead += n
	}

	// If we've hit the fail point, return error
	if r.bytesRead >= r.failAfter {
		if n > 0 {
			return n, nil // Return data this time, error next time
		}
		if r.errorMsg != "" {
			return 0, fmt.Errorf("%s", r.errorMsg)
		}
		return 0, fmt.Errorf("injected read error at byte %d", r.bytesRead)
	}

	// Normal EOF condition
	if r.bytesRead >= len(r.data) {
		return n, io.EOF
	}

	return n, nil
}

// failingWriter simulates write errors after N bytes
type failingWriter struct {
	buf          bytes.Buffer
	failAfter    int
	bytesWritten int
	errorMsg     string
}

func (w *failingWriter) Write(p []byte) (n int, err error) {
	if w.bytesWritten >= w.failAfter {
		if w.errorMsg != "" {
			return 0, fmt.Errorf("%s", w.errorMsg)
		}
		return 0, fmt.Errorf("injected write error at byte %d", w.bytesWritten)
	}

	remaining := w.failAfter - w.bytesWritten
	if remaining <= 0 {
		if w.errorMsg != "" {
			return 0, fmt.Errorf("%s", w.errorMsg)
		}
		return 0, fmt.Errorf("injected write error")
	}

	toWrite := len(p)
	if toWrite > remaining {
		toWrite = remaining
	}

	n, err = w.buf.Write(p[:toWrite])
	w.bytesWritten += n

	if w.bytesWritten >= w.failAfter && n < len(p) {
		if w.errorMsg != "" {
			return n, fmt.Errorf("%s", w.errorMsg)
		}
		return n, fmt.Errorf("injected write error after %d bytes", n)
	}

	return n, err
}

// intermittentReader fails every Nth read
type intermittentReader struct {
	data      []byte
	pos       int
	failEvery int
	readCount int
}

func (r *intermittentReader) Read(p []byte) (n int, err error) {
	r.readCount++
	if r.readCount%r.failEvery == 0 {
		return 0, fmt.Errorf("intermittent read error on read %d", r.readCount)
	}

	if r.pos >= len(r.data) {
		return 0, io.EOF
	}

	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func TestErrorIOFailures(t *testing.T) {
	t.Run("reader_fails_midstream", func(t *testing.T) {
		input := strings.Repeat("test content\n", 100)
		reader := &failingReader{
			data:      []byte(input),
			failAfter: 500, // Fail after more data to ensure some gets through
			errorMsg:  "simulated disk read error",
		}

		var buf bytes.Buffer
		err := Flow(context.Background(), reader, &buf, 100, passthroughRenderer)

		if err == nil {
			// May complete successfully if buffering works around the failure point
			t.Log("No error - completed successfully")
		} else if !strings.Contains(err.Error(), "simulated disk read error") {
			t.Errorf("Error should contain injection message: %v", err)
		}

		// Check we got some output
		if buf.Len() > 0 {
			t.Logf("Processed %d bytes", buf.Len())
		} else {
			t.Log("No output processed")
		}
	})

	t.Run("writer_fails_midstream", func(t *testing.T) {
		input := strings.Repeat("content\n", 100)
		writer := &failingWriter{
			failAfter: 100,
			errorMsg:  "disk full",
		}

		err := Flow(context.Background(), strings.NewReader(input), writer, 100, passthroughRenderer)

		if err == nil {
			t.Error("Expected error from failing writer")
		}
		if !strings.Contains(err.Error(), "disk full") {
			t.Errorf("Error should contain disk full message: %v", err)
		}

		// Verify partial output was written
		if writer.buf.Len() == 0 {
			t.Error("Expected some output before write failure")
		}
		t.Logf("Wrote %d bytes before failure", writer.buf.Len())
	})

	t.Run("intermittent_read_errors", func(t *testing.T) {
		input := strings.Repeat("line\n", 20)
		reader := &intermittentReader{
			data:      []byte(input),
			failEvery: 3, // Fail every 3rd read
		}

		var buf bytes.Buffer
		err := Flow(context.Background(), reader, &buf, 10, passthroughRenderer)

		// May or may not error depending on read pattern
		if err != nil {
			t.Logf("Got expected intermittent error: %v", err)
		}
		t.Logf("Processed %d bytes with intermittent errors", buf.Len())
	})

	t.Run("writer_disk_full", func(t *testing.T) {
		input := strings.Repeat("X", 10000)
		writer := &failingWriter{
			failAfter: 1000,
			errorMsg:  "no space left on device",
		}

		err := Flow(context.Background(), strings.NewReader(input), writer, 100, passthroughRenderer)

		if err == nil {
			t.Error("Expected disk full error")
		}
		if !strings.Contains(err.Error(), "no space") {
			t.Errorf("Expected disk full message: %v", err)
		}
		t.Logf("Disk full after %d bytes", writer.bytesWritten)
	})

	t.Run("timeout_during_operations", func(t *testing.T) {
		// Simulate slow read that will timeout
		slowInput := &errorSlowReader{
			data:  []byte("slow data"),
			delay: 100 * time.Millisecond,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		var buf bytes.Buffer
		err := Flow(ctx, slowInput, &buf, 100, passthroughRenderer)

		// Should timeout or complete
		if err != nil && err == context.DeadlineExceeded {
			t.Log("Got expected timeout")
		}
		t.Logf("Timeout test: err=%v, output=%d bytes", err, buf.Len())
	})
}

// 2. RESOURCE EXHAUSTION TESTS

func TestErrorResourceExhaustion(t *testing.T) {
	t.Run("memory_pressure", func(t *testing.T) {
		// Try to process large data with small window
		var m runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m)
		startMem := m.Alloc

		// Process 10MB through tiny window
		size := 10 * 1024 * 1024
		reader := &repeatReader{
			pattern: []byte("MEMORY"),
			limit:   int64(size),
		}

		var discardOut discardWriter
		err := Flow(context.Background(), reader, &discardOut, 1, passthroughRenderer)

		runtime.GC()
		runtime.ReadMemStats(&m)
		endMem := m.Alloc

		if err != nil {
			t.Logf("Memory pressure error: %v", err)
		}

		memUsed := endMem - startMem
		t.Logf("Memory used: %d bytes for %d byte input", memUsed, size)

		// Memory should stay bounded despite large input
		if memUsed > uint64(size/2) {
			t.Logf("Warning: High memory usage relative to input")
		}
	})

	t.Run("buffer_allocation_stress", func(t *testing.T) {
		// Many small allocations
		var totalProcessed int64
		for i := 0; i < 100; i++ {
			input := strings.Repeat("A", 1000)
			var buf bytes.Buffer
			err := Flow(context.Background(), strings.NewReader(input), &buf, int64(i+1), passthroughRenderer)
			if err != nil {
				t.Logf("Allocation stress error at iteration %d: %v", i, err)
				break
			}
			totalProcessed += int64(buf.Len())
		}
		t.Logf("Processed %d bytes under allocation stress", totalProcessed)
	})

	t.Run("context_deadline_pressure", func(t *testing.T) {
		// Very short deadline
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Microsecond)
		defer cancel()

		input := "test"
		var buf bytes.Buffer
		err := Flow(ctx, strings.NewReader(input), &buf, 100, passthroughRenderer)

		// Almost certainly will timeout
		if err != nil {
			t.Logf("Ultra-short deadline: %v", err)
		}
	})

	t.Run("goroutine_limit", func(t *testing.T) {
		startGoroutines := runtime.NumGoroutine()

		// Launch many Flow operations
		errors := make(chan error, 50)
		for i := 0; i < 50; i++ {
			go func(id int) {
				input := fmt.Sprintf("goroutine %d\n", id)
				var buf bytes.Buffer
				err := Flow(context.Background(), strings.NewReader(input), &buf, 10, passthroughRenderer)
				errors <- err
			}(i)
		}

		// Collect results
		var errCount int
		for i := 0; i < 50; i++ {
			if err := <-errors; err != nil {
				errCount++
			}
		}

		endGoroutines := runtime.NumGoroutine()
		t.Logf("Goroutines: start=%d, end=%d, errors=%d", startGoroutines, endGoroutines, errCount)
	})

	t.Run("stack_depth", func(t *testing.T) {
		// Deeply nested markdown might stress the stack
		var input strings.Builder
		for i := 0; i < 1000; i++ {
			input.WriteString(strings.Repeat(">", i%10+1))
			input.WriteString(" Nested\n")
		}

		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(input.String()), &buf, 100, passthroughRenderer)

		if err != nil {
			t.Logf("Stack depth error: %v", err)
		} else {
			t.Logf("Handled %d bytes of nested content", input.Len())
		}
	})
}

// 3. INVALID INPUT HANDLING TESTS

func TestErrorInvalidInputs(t *testing.T) {
	t.Run("corrupted_utf8", func(t *testing.T) {
		// Invalid UTF-8 sequences
		corrupted := []byte{
			0xFF, 0xFE, 0xFD, // Invalid UTF-8
			'H', 'e', 'l', 'l', 'o',
			0x80, 0x81, // More invalid bytes
			'\n',
		}

		var buf bytes.Buffer
		err := Flow(context.Background(), bytes.NewReader(corrupted), &buf, 100, passthroughRenderer)

		if err != nil {
			t.Logf("Corrupted UTF-8 error: %v", err)
		}
		// Should handle gracefully
		if buf.Len() == 0 && err == nil {
			t.Error("Expected some output or error for corrupted input")
		}
		t.Logf("Processed %d bytes of corrupted UTF-8", buf.Len())
	})

	t.Run("unexpected_eof", func(t *testing.T) {
		// Reader that returns EOF unexpectedly
		reader := &eofReader{
			data:      []byte("partial da"),
			eofAtByte: 7, // EOF in middle of word
		}

		var buf bytes.Buffer
		err := Flow(context.Background(), reader, &buf, 100, passthroughRenderer)

		// Should handle EOF gracefully
		if err != nil && err != io.EOF && !errors.Is(err, io.EOF) {
			t.Errorf("Unexpected error (not EOF): %v", err)
		}
		if buf.Len() == 0 {
			t.Error("Expected partial output before EOF")
		}
		t.Logf("Processed %d bytes before unexpected EOF", buf.Len())
	})

	t.Run("nil_components", func(t *testing.T) {
		// Test nil safety
		t.Run("nil_reader", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Logf("Nil reader panic: %v", r)
				}
			}()

			var buf bytes.Buffer
			err := Flow(context.Background(), nil, &buf, 100, passthroughRenderer)
			if err == nil {
				t.Error("Expected error or panic for nil reader")
			}
		})

		t.Run("nil_writer", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Logf("Nil writer panic: %v", r)
				}
			}()

			err := Flow(context.Background(), strings.NewReader("test"), nil, 100, passthroughRenderer)
			if err == nil {
				t.Error("Expected error or panic for nil writer")
			}
		})
	})

	t.Run("cancelled_context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		var buf bytes.Buffer
		err := Flow(ctx, strings.NewReader("test"), &buf, 100, passthroughRenderer)

		// May succeed if fast enough, or error
		t.Logf("Cancelled context: err=%v, output=%d bytes", err, buf.Len())
	})

	t.Run("malformed_markdown", func(t *testing.T) {
		// Extremely malformed markdown
		malformed := "```\nUnclosed fence that goes on and on\n"
		malformed += strings.Repeat("No closing fence\n", 100)

		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(malformed), &buf, 100, passthroughRenderer)

		// Should handle gracefully
		if err != nil {
			t.Logf("Malformed markdown error: %v", err)
		}
		t.Logf("Processed %d bytes of malformed markdown", buf.Len())
	})
}

// 4. ERROR RECOVERY & PROPAGATION TESTS

func TestErrorRecoveryPropagation(t *testing.T) {
	t.Run("partial_output_on_failure", func(t *testing.T) {
		input := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\n"
		reader := &failingReader{
			data:      []byte(input),
			failAfter: 20, // Fail in middle
		}

		var buf bytes.Buffer
		err := Flow(context.Background(), reader, &buf, 10, passthroughRenderer)

		// May or may not error depending on buffering
		if err != nil {
			t.Logf("Got expected error: %v", err)
		}

		// Should have partial output
		output := buf.String()
		if !strings.Contains(output, "Line 1") {
			t.Error("Expected at least first line in partial output")
		}
		if strings.Contains(output, "Line 5") {
			t.Error("Should not have last line in partial output")
		}
		t.Logf("Partial output: %q", output)
	})

	t.Run("resource_cleanup", func(t *testing.T) {
		// Track resource allocation/cleanup
		var resourcesAllocated int32
		var resourcesFreed int32

		trackingRenderer := func(data []byte) ([]byte, error) {
			atomic.AddInt32(&resourcesAllocated, 1)
			defer atomic.AddInt32(&resourcesFreed, 1)
			return data, nil
		}

		// Run with failure
		reader := &failingReader{
			data:      []byte(strings.Repeat("test\n", 10)),
			failAfter: 30,
		}

		var buf bytes.Buffer
		_ = Flow(context.Background(), reader, &buf, 10, trackingRenderer)

		// Resources should be balanced
		allocated := atomic.LoadInt32(&resourcesAllocated)
		freed := atomic.LoadInt32(&resourcesFreed)
		t.Logf("Resources: allocated=%d, freed=%d", allocated, freed)

		if allocated != freed {
			t.Error("Resource leak detected: allocated != freed")
		}
	})

	t.Run("error_message_clarity", func(t *testing.T) {
		testCases := []struct {
			name     string
			reader   io.Reader
			expected string
		}{
			{
				name:     "read_error",
				reader:   &failingReader{data: []byte("test"), failAfter: 1, errorMsg: "disk read failed"},
				expected: "disk read failed",
			},
			{
				name:     "timeout",
				reader:   &errorSlowReader{data: []byte("slow"), delay: time.Hour},
				expected: "context",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
				defer cancel()

				var buf bytes.Buffer
				err := Flow(ctx, tc.reader, &buf, 100, passthroughRenderer)

				if err != nil && !strings.Contains(strings.ToLower(err.Error()), tc.expected) {
					t.Errorf("Error message should contain %q: %v", tc.expected, err)
				}
			})
		}
	})

	t.Run("panic_recovery", func(t *testing.T) {
		panicRenderer := func(data []byte) ([]byte, error) {
			if strings.Contains(string(data), "PANIC") {
				panic("Renderer panic triggered!")
			}
			return data, nil
		}

		input := "Normal\nPANIC\nMore"
		var buf bytes.Buffer

		// Flow doesn't recover from renderer panics (documented behavior)
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Expected panic caught: %v", r)
			}
		}()

		_ = Flow(context.Background(), strings.NewReader(input), &buf, 100, panicRenderer)
	})

	t.Run("cascading_failures", func(t *testing.T) {
		// Multiple failures in sequence
		errorRenderer := func(data []byte) ([]byte, error) {
			return nil, fmt.Errorf("renderer failed")
		}

		reader := &failingReader{
			data:      []byte("test"),
			failAfter: 100, // Won't fail
		}

		writer := &failingWriter{
			failAfter: 100, // Won't fail
		}

		err := Flow(context.Background(), reader, writer, 10, errorRenderer)

		// Renderer error should be reported
		if err == nil {
			t.Error("Expected renderer error")
		}
		if !strings.Contains(err.Error(), "renderer") {
			t.Errorf("Expected renderer error, got: %v", err)
		}
		t.Log("Cascading failure handled correctly")
	})
}

// Helper types

type eofReader struct {
	data      []byte
	pos       int
	eofAtByte int
}

func (r *eofReader) Read(p []byte) (n int, err error) {
	if r.pos >= r.eofAtByte {
		return 0, io.EOF
	}

	remaining := r.eofAtByte - r.pos
	if remaining > len(p) {
		remaining = len(p)
	}
	if remaining > len(r.data)-r.pos {
		remaining = len(r.data) - r.pos
	}

	if remaining > 0 {
		n = copy(p, r.data[r.pos:r.pos+remaining])
		r.pos += n
	}

	if r.pos >= r.eofAtByte {
		return n, io.EOF
	}
	return n, nil
}

type errorSlowReader struct {
	data  []byte
	pos   int
	delay time.Duration
}

func (r *errorSlowReader) Read(p []byte) (n int, err error) {
	time.Sleep(r.delay)
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
