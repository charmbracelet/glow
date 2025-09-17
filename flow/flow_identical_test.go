package flow

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestIdenticalOutputAcrossFlowModes(t *testing.T) {
	// Test input that might trigger different behavior
	input := `# Test Document

This is a paragraph with some content.

## Section 1

More content here with **bold** and *italic* text.

` + "```go" + `
func main() {
    fmt.Println("Hello, World!")
}
` + "```" + `

## Section 2

Final paragraph with some text.
`

	// CRITICAL: Generate expected output directly from glamour, not from flow
	expectedOutput := generateGlamourReference(t, input)
	expectedHash := fmt.Sprintf("%x", md5.Sum([]byte(expectedOutput)))

	// Test all flow modes
	modes := []struct {
		name   string
		window int64
	}{
		{"unbuffered", Unbuffered},
		{"buffered", Buffered},
		{"windowed-1", 1},
		{"windowed-1024", 1024},
		{"windowed-4096", 4096},
	}

	outputs := make(map[string]string)
	hashes := make(map[string]string)

	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			var output bytes.Buffer
			err := Flow(
				context.Background(),
				strings.NewReader(input),
				&output,
				mode.window,
				passthroughRenderer, // Use the real glamour renderer, not identity
			)
			if err != nil {
				t.Fatalf("Flow failed for mode %s: %v", mode.name, err)
			}

			result := output.String()
			hash := fmt.Sprintf("%x", md5.Sum([]byte(result)))

			outputs[mode.name] = result
			hashes[mode.name] = hash

			// Compare directly against glamour reference
			if result != expectedOutput {
				t.Errorf("Mode %s produces different output than direct glamour rendering", mode.name)
				t.Errorf("Expected hash: %s", expectedHash)
				t.Errorf("Got hash: %s", hash)

				// Use cmp for better diff output
				if diff := cmp.Diff(expectedOutput, result); diff != "" {
					t.Errorf("Output mismatch (-want +got):\n%s", diff)
				}
			}

			t.Logf("Mode %s: hash=%s, len=%d, matches_glamour=%v",
				mode.name, hash, len(result), result == expectedOutput)
		})
	}

	// Verify all outputs match the glamour reference
	for name, output := range outputs {
		if output != expectedOutput {
			t.Errorf("Mode %s does not match direct glamour output", name)

			// Show detailed difference
			if len(output) != len(expectedOutput) {
				t.Errorf("Length difference: got %d, want %d", len(output), len(expectedOutput))
			}

			// Find first difference
			for i := 0; i < len(output) && i < len(expectedOutput); i++ {
				if output[i] != expectedOutput[i] {
					start := i - 10
					if start < 0 {
						start = 0
					}
					end := i + 10
					if end > len(output) {
						end = len(output)
					}
					if end > len(expectedOutput) {
						end = len(expectedOutput)
					}
					t.Errorf("First difference at byte %d:", i)
					t.Errorf("Expected: %q", expectedOutput[start:end])
					t.Errorf("Got: %q", output[start:end])
					break
				}
			}
		}
	}
}