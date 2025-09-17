package flow

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

// TestStreamingProgressiveOutput tests that output is streamed progressively
func TestStreamingProgressiveOutput(t *testing.T) {
	t.Run("aggressive flush mode", func(t *testing.T) {
		ctx := context.Background()
		r := strings.NewReader("# First\n\n# Second\n\n# Third")

		// Capture output timing
		w := &timedWriter{
			writes: make([]writeEvent, 0),
		}

		err := Flow(ctx, r, w, Unbuffered, passthroughRenderer)
		if err != nil {
			t.Fatalf("Flow error: %v", err)
		}

		// Should have multiple writes for aggressive flush
		if len(w.writes) < 2 {
			t.Errorf("Expected multiple writes for aggressive flush, got %d", len(w.writes))
		}
	})

	t.Run("no flush mode", func(t *testing.T) {
		ctx := context.Background()
		r := strings.NewReader("# First\n\n# Second\n\n# Third")

		w := &timedWriter{
			writes: make([]writeEvent, 0),
		}

		err := Flow(ctx, r, w, Buffered, passthroughRenderer)
		if err != nil {
			t.Fatalf("Flow error: %v", err)
		}

		// Should have single write for no flush mode
		if len(w.writes) != 1 {
			t.Errorf("Expected single write for no flush mode, got %d", len(w.writes))
		}
	})
}

// TestStreamingBufferBoundaries tests buffer boundary handling
func TestStreamingBufferBoundaries(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		window int64
	}{
		{
			name:   "code block at boundary",
			input:  "Header\n```\ncode line 1\ncode line 2\n```\nFooter",
			window: 20, // Forces split in middle
		},
		{
			name:   "paragraph boundary",
			input:  "Para 1\n\nPara 2\n\nPara 3",
			window: 10,
		},
		{
			name:   "header boundary",
			input:  "# Header 1\nContent\n## Header 2\nMore",
			window: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate expected output directly from glamour
			expectedOutput := generateGlamourReference(t, tt.input)

			// Test with different window sizes - output should match glamour
			ctx := context.Background()

			// First with small window
			r1 := strings.NewReader(tt.input)
			var buf1 bytes.Buffer
			err := Flow(ctx, r1, &buf1, tt.window, passthroughRenderer)
			if err != nil {
				t.Fatalf("Flow error with small window: %v", err)
			}

			// Then with no window
			r2 := strings.NewReader(tt.input)
			var buf2 bytes.Buffer
			err = Flow(ctx, r2, &buf2, Buffered, passthroughRenderer)
			if err != nil {
				t.Fatalf("Flow error with no window: %v", err)
			}

			// Both should match glamour reference
			if buf1.String() != expectedOutput {
				if diff := cmp.Diff(expectedOutput, buf1.String()); diff != "" {
					t.Errorf("Small window output mismatch (-want +got):\n%s", diff)
				}
			}

			if buf2.String() != expectedOutput {
				if diff := cmp.Diff(expectedOutput, buf2.String()); diff != "" {
					t.Errorf("Buffered output mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

// TestStreamingMemoryBounds tests memory usage bounds
func TestStreamingMemoryBounds(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	t.Run("huge single line", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Create 10MB single line
		r := &repeatReader{
			pattern: []byte("A"),
			limit:   10 * 1024 * 1024,
		}
		var buf discardWriter

		err := Flow(ctx, r, &buf, 4096, passthroughRenderer)
		if err != nil && err != context.DeadlineExceeded {
			t.Fatalf("Flow error: %v", err)
		}

		// Should have processed significant data
		if buf.written < 1024*1024 {
			t.Errorf("Expected to process at least 1MB, got %d bytes", buf.written)
		}
	})

	t.Run("resource limit enforcement", func(t *testing.T) {
		// This test verifies the maxActiveReads limit
		ctx := context.Background()

		// Create a reader that tracks concurrent reads
		r := &concurrentTrackingReader{
			data: []byte(strings.Repeat("Test data\n", 1000)),
		}

		var buf bytes.Buffer
		renderer := func(data []byte) ([]byte, error) {
			return data, nil
		}

		err := Flow(ctx, r, &buf, Unbuffered, renderer)
		if err != nil {
			t.Fatalf("Flow error: %v", err)
		}

		// Should never exceed maxActiveReads (2)
		if r.maxConcurrent > 2 {
			t.Errorf("Exceeded max concurrent reads: %d > 2", r.maxConcurrent)
		}
	})
}

// TestStreamingSignalHandling tests signal handling during streaming
func TestStreamingSignalHandling(t *testing.T) {
	signals := []struct {
		name        string
		createError func() error
	}{
		{
			name:        "SIGPIPE",
			createError: func() error { return io.ErrClosedPipe },
		},
		{
			name:        "context cancelled",
			createError: func() error { return context.Canceled },
		},
		{
			name:        "deadline exceeded",
			createError: func() error { return context.DeadlineExceeded },
		},
	}

	for _, sig := range signals {
		t.Run(sig.name, func(t *testing.T) {
			ctx := context.Background()
			r := strings.NewReader("# Test content")

			// Writer that returns specific error
			w := &errorWriter{
				err:        sig.createError(),
				afterBytes: 5,
			}

			renderer := func(data []byte) ([]byte, error) {
				return data, nil
			}

			err := Flow(ctx, r, w, Unbuffered, renderer)
			// These errors should be suppressed
			if err != nil {
				t.Errorf("Expected error to be suppressed, got: %v", err)
			}
		})
	}
}

// TestStreamingLatency tests first-byte latency
func TestStreamingLatency(t *testing.T) {
	t.Run("first byte latency", func(t *testing.T) {
		ctx := context.Background()

		// Reader that delays after first chunk
		r := &slowReader{
			chunks: []string{"# First\n", "# Second\n"},
			delay:  500 * time.Millisecond,
		}

		w := &timedWriter{
			writes: make([]writeEvent, 0),
		}

		renderer := func(data []byte) ([]byte, error) {
			return data, nil
		}

		start := time.Now()
		go func() {
			_ = Flow(ctx, r, w, Unbuffered, renderer)
		}()

		// Wait for first write
		timeout := time.After(100 * time.Millisecond)
		for {
			select {
			case <-timeout:
				t.Fatalf("First byte took too long (>100ms)")
			default:
				writes := w.getWrites()
				if len(writes) > 0 {
					elapsed := writes[0].timestamp.Sub(start)
					if elapsed > 100*time.Millisecond {
						t.Errorf("First byte latency too high: %v", elapsed)
					}
					return
				}
				time.Sleep(5 * time.Millisecond)
			}
		}
	})
}

// Helper types for streaming tests

type writeEvent struct {
	data      []byte
	timestamp time.Time
}

type timedWriter struct {
	mu     sync.Mutex
	writes []writeEvent
}

func (w *timedWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	data := make([]byte, len(p))
	copy(data, p)
	w.writes = append(w.writes, writeEvent{
		data:      data,
		timestamp: time.Now(),
	})
	return len(p), nil
}

func (w *timedWriter) getWrites() []writeEvent {
	w.mu.Lock()
	defer w.mu.Unlock()
	result := make([]writeEvent, len(w.writes))
	copy(result, w.writes)
	return result
}

type concurrentTrackingReader struct {
	mu            sync.Mutex
	data          []byte
	pos           int
	concurrent    int
	maxConcurrent int
}

func (r *concurrentTrackingReader) Read(p []byte) (n int, err error) {
	r.mu.Lock()
	r.concurrent++
	if r.concurrent > r.maxConcurrent {
		r.maxConcurrent = r.concurrent
	}
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		r.concurrent--
		r.mu.Unlock()
	}()

	// Simulate some read time
	time.Sleep(time.Millisecond)

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.pos >= len(r.data) {
		return 0, io.EOF
	}

	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

type errorWriter struct {
	err        error
	afterBytes int
	written    int
}

func (w *errorWriter) Write(p []byte) (n int, err error) {
	if w.written >= w.afterBytes {
		return 0, w.err
	}
	w.written += len(p)
	return len(p), nil
}
