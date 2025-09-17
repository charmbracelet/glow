package flow

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

// TestFlowPerformanceLargeBuffer tests that buffer scanning is O(n) not O(n²)
func TestFlowPerformanceLargeBuffer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create a large input with no safe boundaries (worst case for O(n²))
	// This is a single massive code block that won't have boundaries
	var input strings.Builder
	input.WriteString("```\n")
	for i := 0; i < 500000; i++ { // 500K lines inside code block
		input.WriteString("code line without any safe boundaries\n")
	}
	input.WriteString("```\n")

	ctx := context.Background()
	r := strings.NewReader(input.String())
	var buf bytes.Buffer

	// Measure time for processing
	start := time.Now()
	err := Flow(ctx, r, &buf, 4096, passthroughRenderer)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Flow error: %v", err)
	}

	// With O(n²) behavior, this would take many seconds
	// With O(n) behavior, should complete reasonably quickly
	// Allow 5 seconds for slower test environments and glamour processing
	if elapsed > 5*time.Second {
		t.Errorf("Processing took too long (%v), possible O(n²) behavior", elapsed)
	}

	t.Logf("Processed %d bytes in %v", input.Len(), elapsed)
}

// TestFlowIncrementalBoundarySearch tests that boundaries are found incrementally
func TestFlowIncrementalBoundarySearch(t *testing.T) {
	// Create a custom FlowBuffer to test incremental search
	config := Config{
		Window: 4096,
		Render: func(data []byte) ([]byte, error) {
			return data, nil
		},
	}

	fb, err := NewBuffer(config, io.Discard)
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}

	// Add initial data without boundary
	fb.accum = append(fb.accum, []byte("Initial content without boundary")...)

	// Test findSafeBoundary function

	// First check - no boundary
	boundary := fb.findSafeBoundary()
	if boundary > 0 {
		t.Errorf("Should not have boundary in initial content")
	}

	// Add data with boundary
	fb.accum = append(fb.accum, []byte("\n\nParagraph after boundary")...)

	// Should find boundary now
	boundary = fb.findSafeBoundary()
	if boundary <= 0 {
		t.Errorf("Should have found paragraph boundary")
	}

}

// BenchmarkFlowLargeInput benchmarks large input processing
func BenchmarkFlowLargeInput(b *testing.B) {
	// Create 1MB of markdown content
	var input strings.Builder
	for i := 0; i < 10000; i++ {
		input.WriteString("# Header\n\nParagraph content with some text.\n\n")
	}
	inputStr := input.String()

	ctx := context.Background()
	renderer := func(data []byte) ([]byte, error) {
		return data, nil
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		r := strings.NewReader(inputStr)
		var buf bytes.Buffer
		_ = Flow(ctx, r, &buf, 4096, renderer)
	}

	b.SetBytes(int64(len(inputStr)))
}
