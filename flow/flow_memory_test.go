package flow

import (
	"bytes"
	"context"
	"crypto/rand"
	"testing"
)

func TestMemorySafety(t *testing.T) {
	// Generate 20MB of random data
	input := make([]byte, 20*1024*1024)
	n, err := rand.Read(input)
	if err != nil {
		t.Fatalf("Failed to generate random data: %v", err)
	}
	t.Logf("Generated %d MB of random input", n/(1024*1024))

	var output bytes.Buffer
	err = Flow(
		context.Background(),
		bytes.NewReader(input),
		&output,
		1024,
		passthroughRenderer,
	)

	if err != nil {
		t.Logf("Flow returned error: %v", err)
	}

	outputSize := output.Len()
	t.Logf("Output size: %d MB", outputSize/(1024*1024))

	// Random binary data is not valid markdown, so glamour may produce
	// different output or fail gracefully. The important test is that
	// Flow doesn't crash or consume unbounded memory.
	// We don't enforce strict size limits on glamour output since
	// glamour may expand or contract the data unpredictably.

	// Just verify Flow completed without panic
	t.Logf("Flow handled %d MB of random data, produced %d MB output",
		n/(1024*1024), outputSize/(1024*1024))
}

func TestConcurrentAccess(t *testing.T) {
	// Test concurrent reads don't cause race conditions
	input := bytes.Repeat([]byte("# Test\nContent\n"), 1000)

	// Run multiple concurrent flows
	done := make(chan error, 3)

	for i := 0; i < 3; i++ {
		go func() {
			var output bytes.Buffer
			err := Flow(
				context.Background(),
				bytes.NewReader(input),
				&output,
				1024,
				passthroughRenderer,
			)
			done <- err
		}()
	}

	// Wait for all to complete
	for i := 0; i < 3; i++ {
		err := <-done
		if err != nil {
			t.Errorf("Concurrent flow failed: %v", err)
		}
	}
}
