package flow

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/google/go-cmp/cmp"
)

// Compare outputs for equivalence
func compareOutputs(t *testing.T, expected, actual []byte) {
	if !bytes.Equal(expected, actual) {
		t.Errorf("Output mismatch\nExpected: %q\nActual: %q", expected, actual)
	}
}

// generateGlamourReference creates the expected output directly from glamour
// This is the SINGLE SOURCE OF TRUTH for all test comparisons
func generateGlamourReference(t *testing.T, input string) string {
	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath("notty"),
		glamour.WithWordWrap(0), // Critical: disable word wrap for consistency
	)
	if err != nil {
		t.Fatalf("Failed to create glamour renderer: %v", err)
	}

	output, err := r.RenderBytes([]byte(input))
	if err != nil {
		t.Fatalf("Glamour rendering failed: %v", err)
	}

	return string(output)
}

// assertFlowMatchesGlamour validates that flow output matches direct glamour rendering
func assertFlowMatchesGlamour(t *testing.T, input string, window int64) {
	expected := generateGlamourReference(t, input)

	var output bytes.Buffer
	err := Flow(context.Background(),
		strings.NewReader(input),
		&output,
		window,
		passthroughRenderer)
	if err != nil {
		t.Fatalf("Flow failed with window=%d: %v", window, err)
	}

	if diff := cmp.Diff(expected, output.String()); diff != "" {
		t.Errorf("Flow output mismatch for window=%d (-want +got):\n%s",
			window, diff)
	}
}

// Helper to create a simple pass-through renderer for tests
func passthroughRenderer(data []byte) ([]byte, error) {
	// Use real glamour for consistent testing
	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath("notty"),
		glamour.WithWordWrap(0), // No word wrapping for consistency
	)
	if err != nil {
		return nil, err
	}

	return r.RenderBytes(data)
}

// Run flow with specified window and return output
func runFlow(t *testing.T, input string, window int64) string {
	ctx := context.Background()
	var buf bytes.Buffer
	err := Flow(ctx, strings.NewReader(input), &buf, window, passthroughRenderer)
	if err != nil {
		t.Fatalf("Flow failed: %v", err)
	}
	return buf.String()
}

// Run flow with timeout and return output
func runFlowWithTimeout(t *testing.T, input string, window int64, timeout time.Duration) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var buf bytes.Buffer
	err := Flow(ctx, strings.NewReader(input), &buf, window, passthroughRenderer)
	if err != nil && err != context.DeadlineExceeded {
		t.Fatalf("Flow failed: %v", err)
	}
	return buf.String()
}

// repeatReader generates a repeating pattern without pre-allocating memory
type repeatReader struct {
	pattern []byte
	limit   int64
	read    int64
}

func (r *repeatReader) Read(p []byte) (n int, err error) {
	if r.read >= r.limit {
		return 0, io.EOF
	}

	remaining := r.limit - r.read
	toRead := int64(len(p))
	if toRead > remaining {
		toRead = remaining
	}

	for i := int64(0); i < toRead; i++ {
		p[i] = r.pattern[(r.read+i)%int64(len(r.pattern))]
	}

	n = int(toRead)
	r.read += toRead
	return n, nil
}

// discardWriter counts bytes written but discards the data
type discardWriter struct {
	written int64
}

func (w *discardWriter) Write(p []byte) (int, error) {
	w.written += int64(len(p))
	return len(p), nil
}
