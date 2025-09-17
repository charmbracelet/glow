package flow

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestBufferTorture(t *testing.T) {
	// Generate 15MB pathological input
	var buf bytes.Buffer

	// Add deeply nested fences
	for i := 0; i < 100; i++ {
		buf.WriteString(strings.Repeat("`", (i%10)+3) + "\n")
		buf.WriteString("code\n")
	}

	// Add 5MB single line
	buf.WriteString(strings.Repeat("x", 5*1024*1024))
	buf.WriteString("\n")

	// Close fences
	for i := 99; i >= 0; i-- {
		buf.WriteString(strings.Repeat("`", (i%10)+3) + "\n")
	}

	// Exceed 15MB
	buf.WriteString(strings.Repeat("overflow", 2*1024*1024))

	t.Logf("Input size: %d MB", buf.Len()/(1024*1024))

	var output bytes.Buffer
	err := Flow(
		context.Background(),
		&buf,
		&output,
		1024,
		passthroughRenderer,
	)

	if err != nil {
		t.Logf("Flow returned error (expected): %v", err)
	}

	t.Logf("Output size: %d bytes", output.Len())

	// Verify no panic occurred and output is roughly bounded
	// Allow small overhead for glamour formatting (newlines, etc)
	maxAllowed := Windowed + 1024 // Allow 1KB overhead for glamour
	if output.Len() > maxAllowed {
		t.Errorf("Output exceeded reasonable bounds: %d > %d", output.Len(), maxAllowed)
	}
}
