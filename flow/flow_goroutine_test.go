package flow

import (
	"bytes"
	"context"
	"io"
	"runtime"
	"testing"
	"time"
)

// TestGoroutineDoSPrevention verifies that streaming does not spawn unbounded goroutines
func TestGoroutineDoSPrevention(t *testing.T) {
	// Get baseline goroutine count
	runtime.GC()
	time.Sleep(10 * time.Millisecond)
	baseline := runtime.NumGoroutine()
	t.Logf("Baseline goroutine count: %d", baseline)

	// Create a high-speed reader that simulates 1GB/s stream
	ctx := context.Background()
	r := &highSpeedReader{
		data:      []byte("# Test data\n\nContent here\n\n"),
		totalSize: 100 * 1024 * 1024, // 100MB
	}
	var buf bytes.Buffer

	// Start Flow in goroutine and monitor goroutine count
	done := make(chan error, 1)
	go func() {
		done <- Flow(ctx, r, &buf, 4096, passthroughRenderer)
	}()

	// Monitor goroutine count during streaming
	maxGoroutines := baseline
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(2 * time.Second)
	monitoring := true

	for monitoring {
		select {
		case <-ticker.C:
			current := runtime.NumGoroutine()
			if current > maxGoroutines {
				maxGoroutines = current
			}
			// Critical: Should never exceed baseline + 3
			// (1 for Flow, 1 for reader goroutine, 1 for test goroutine)
			if current > baseline+5 {
				t.Errorf("Goroutine leak detected: %d goroutines (baseline: %d)", current, baseline)
				monitoring = false
			}
		case err := <-done:
			if err != nil {
				t.Errorf("Flow error: %v", err)
			}
			monitoring = false
		case <-timeout:
			t.Log("Test timed out (expected for continuous stream)")
			monitoring = false
		}
	}

	t.Logf("Max goroutines during test: %d (baseline: %d)", maxGoroutines, baseline)

	// Verify we stayed within bounds
	if maxGoroutines > baseline+5 {
		t.Fatalf("Goroutine DoS vulnerability: max %d exceeded baseline+5 (%d)", maxGoroutines, baseline+5)
	}
}

// TestPagerResponseTime verifies sub-100ms response when pager exits
func TestPagerResponseTime(t *testing.T) {
	// Create a slow reader
	r := &slowStreamReader{
		chunks: []string{"# First\n", "## Second\n", "### Third\n"},
		delay:  1 * time.Second, // 1 second between chunks
	}
	var buf bytes.Buffer

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Start Flow
	done := make(chan error, 1)
	go func() {
		done <- Flow(ctx, r, &buf, Unbuffered, passthroughRenderer)
	}()

	// Wait for first chunk to be processed
	time.Sleep(50 * time.Millisecond)

	// Cancel context (simulating pager exit)
	start := time.Now()
	cancel()

	// Wait for Flow to exit
	select {
	case <-done:
		elapsed := time.Since(start)
		if elapsed > 100*time.Millisecond {
			t.Errorf("Flow took %v to respond to cancellation (must be <100ms)", elapsed)
		} else {
			t.Logf("Flow responded in %v", elapsed)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Flow did not respond to cancellation within 200ms")
	}
}

// highSpeedReader simulates a high-speed data stream
type highSpeedReader struct {
	data      []byte
	totalSize int
	sent      int
}

func (r *highSpeedReader) Read(p []byte) (n int, err error) {
	if r.sent >= r.totalSize {
		return 0, io.EOF
	}

	// Always fill the buffer (simulating high-speed stream)
	for n < len(p) && r.sent < r.totalSize {
		toCopy := len(r.data)
		if n+toCopy > len(p) {
			toCopy = len(p) - n
		}
		if r.sent+toCopy > r.totalSize {
			toCopy = r.totalSize - r.sent
		}
		copy(p[n:n+toCopy], r.data[:toCopy])
		n += toCopy
		r.sent += toCopy
	}

	return n, nil
}

// slowStreamReader provides data with delays
type slowStreamReader struct {
	chunks []string
	idx    int
	delay  time.Duration
}

func (r *slowStreamReader) Read(p []byte) (n int, err error) {
	if r.idx >= len(r.chunks) {
		return 0, io.EOF
	}

	if r.idx > 0 {
		time.Sleep(r.delay)
	}

	n = copy(p, r.chunks[r.idx])
	r.idx++
	return n, nil
}
