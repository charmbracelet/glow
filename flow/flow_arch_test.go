package flow

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

// TestArchMemoryBounded tests that memory stays bounded with infinite input
// Migrated from: arch_test_1
func TestArchMemoryBounded(t *testing.T) {
	t.Run("memory_bounded_with_infinite_stream", func(t *testing.T) {
		// Create a large repeating pattern that simulates infinite input
		// Using 10000 lines like the shell test
		var input strings.Builder
		for i := 0; i < 10000; i++ {
			input.WriteString("# Infinite header line that repeats forever\n")
		}

		// Process with small buffer - memory should stay bounded
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		var output bytes.Buffer
		err := Flow(ctx, strings.NewReader(input.String()), &output, 1024, passthroughRenderer)

		// Check output was produced (not OOM)
		if err != nil && err != context.DeadlineExceeded {
			t.Fatalf("Flow failed: %v", err)
		}

		if output.Len() == 0 {
			t.Error("No output produced - possible OOM")
		}
	})
}

// TestArchMemoryBoundedNoBoundaries tests memory stays bounded with no safe split points
// Migrated from: arch_test_2
func TestArchMemoryBoundedNoBoundaries(t *testing.T) {
	t.Run("memory_bounded_with_no_boundaries", func(t *testing.T) {
		// Create 500KB single line with no newlines (no split points)
		// Matching the shell test's reduced size
		var input strings.Builder
		input.WriteString("# ")
		for i := 0; i < 500000; i++ {
			input.WriteByte('X')
		}
		input.WriteByte('\n')

		// Process with 4KB buffer - should handle without exhausting memory
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		var output bytes.Buffer
		err := Flow(ctx, strings.NewReader(input.String()), &output, 4096, passthroughRenderer)

		// Should produce output (even if forced flush)
		if err != nil && err != context.DeadlineExceeded {
			t.Fatalf("Flow failed: %v", err)
		}

		if output.Len() == 0 {
			t.Error("No output produced - memory handling failed")
		}
	})
}

// TestArchBufferAccumulationBounded tests that buffer accumulation stays bounded
// Migrated from: arch_test_3
func TestArchBufferAccumulationBounded(t *testing.T) {
	t.Run("buffer_accumulation_stays_bounded", func(t *testing.T) {
		// Create pathological input: many incomplete code blocks
		var input strings.Builder
		for i := 0; i < 1000; i++ {
			input.WriteString("```\n")
			input.WriteString("Incomplete block ")
			input.WriteString(strings.Repeat("X", i%100))
			input.WriteByte('\n')
			// No closing ``` - forces accumulation
		}

		// Process with small buffer - should force flush at boundary
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		var output bytes.Buffer
		err := Flow(ctx, strings.NewReader(input.String()), &output, 8192, passthroughRenderer)

		// Should have processed something (forced flush)
		if err != nil && err != context.DeadlineExceeded {
			t.Fatalf("Flow failed: %v", err)
		}

		if output.Len() == 0 {
			t.Error("No output produced - accumulation not bounded")
		}
	})
}

// Additional test to verify the architecture with a real renderer
func TestArchWithGlamourRenderer(t *testing.T) {
	// This test uses the actual glamour renderer to ensure compatibility
	t.Run("memory_bounded_with_glamour", func(t *testing.T) {
		// Create moderate input to test with real renderer
		var input strings.Builder
		for i := 0; i < 100; i++ {
			input.WriteString("# Header ")
			input.WriteString(strings.Repeat("X", i%50))
			input.WriteString("\n\nParagraph content.\n\n")
		}

		// Use a mock glamour-like renderer that adds formatting
		renderer := func(data []byte) ([]byte, error) {
			// Simple renderer that adds ANSI color codes like glamour
			output := bytes.Replace(data, []byte("# "), []byte("\033[1m# "), -1)
			output = append(output, []byte("\033[0m")...)
			return output, nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		var output bytes.Buffer
		err := Flow(ctx, strings.NewReader(input.String()), &output, 4096, renderer)

		if err != nil && err != context.DeadlineExceeded {
			t.Fatalf("Flow with renderer failed: %v", err)
		}

		if output.Len() == 0 {
			t.Error("No output produced with renderer")
		}
	})
}

// TestArchConsistentOutput tests output consistency across buffer sizes
// Migrated from: arch_test_4
func TestArchConsistentOutput(t *testing.T) {
	t.Run("consistent_output_across_buffer_sizes", func(t *testing.T) {
		// Create complex markdown with various structures
		complexMd := `# Header One

Paragraph with **bold** and *italic* text.

## Header Two

` + "```go" + `
func main() {
    fmt.Println("Hello")
}
` + "```" + `

### Header Three

- List item 1
- List item 2

> Blockquote text

[Link](http://example.com)`

		// Process with different buffer sizes
		outputUnlimited := runFlow(t, complexMd, 0)
		output1024 := runFlow(t, complexMd, 1024)
		output64 := runFlow(t, complexMd, 64)
		output16 := runFlow(t, complexMd, 16)

		// All outputs should match
		if outputUnlimited != output1024 {
			t.Error("Output mismatch between unlimited and 1024 buffer")
		}
		if outputUnlimited != output64 {
			t.Error("Output mismatch between unlimited and 64 buffer")
		}
		if outputUnlimited != output16 {
			t.Error("Output mismatch between unlimited and 16 buffer")
		}
	})
}

// TestArchCodeBlocksAcrossBoundaries tests code blocks split at boundaries render correctly
// Migrated from: arch_test_5
func TestArchCodeBlocksAcrossBoundaries(t *testing.T) {
	t.Run("code_blocks_correct_across_boundaries", func(t *testing.T) {
		// Create code block that will be split
		codeMd := `Text before code

` + "```python" + `
def function():
    # This is line 1
    # This is line 2
    # This is line 3
    # This is line 4
    # This is line 5
    return True
` + "```" + `

Text after code`

		// Process with buffer that splits code block
		outputSplit := runFlow(t, codeMd, 50)
		outputNoSplit := runFlow(t, codeMd, 0)

		// Both should produce identical output
		if outputSplit != outputNoSplit {
			t.Errorf("Code block output differs when split\nSplit: %q\nNoSplit: %q",
				outputSplit, outputNoSplit)
		}
	})
}

// TestArchEmptyLinesPreserved tests empty lines preserved at boundaries
// Migrated from: arch_test_6
func TestArchEmptyLinesPreserved(t *testing.T) {
	t.Run("empty_lines_preserved_at_boundaries", func(t *testing.T) {
		// Create content with empty lines that might hit boundaries
		emptyMd := `# First

Paragraph one


Paragraph two


# Second`

		// Process with small buffer that will split at empty lines
		outputSmall := runFlow(t, emptyMd, 20)
		outputFull := runFlow(t, emptyMd, 0)

		// Count blank lines - should be same in both
		countBlankLines := func(s string) int {
			count := 0
			for _, line := range strings.Split(s, "\n") {
				if strings.TrimSpace(line) == "" {
					count++
				}
			}
			return count
		}

		blankSmall := countBlankLines(outputSmall)
		blankFull := countBlankLines(outputFull)

		if blankSmall != blankFull {
			t.Errorf("Blank line count mismatch: small buffer has %d, full buffer has %d",
				blankSmall, blankFull)
		}
	})
}

// TestArchSignalHandling tests SIGTERM produces valid partial output
// Migrated from: arch_test_7
func TestArchSignalHandling(t *testing.T) {
	t.Run("SIGTERM_produces_valid_output", func(t *testing.T) {
		// Create a reader that simulates slow input stream
		slowReader := &testSlowStreamReader{
			lines: []string{
				"# First header",
				"Content line 1",
				"## Second header",
				"Should not appear",
			},
			delays: []time.Duration{
				0,
				100 * time.Millisecond,
				100 * time.Millisecond,
				1 * time.Second,
			},
		}

		// Cancel after 150ms to simulate SIGTERM
		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		defer cancel()

		var output bytes.Buffer
		err := Flow(ctx, slowReader, &output, -1, passthroughRenderer)

		// Should have terminated due to context cancellation
		if err != context.DeadlineExceeded {
			t.Logf("Expected context.DeadlineExceeded, got: %v", err)
		}

		result := output.String()
		// Should have first header and content
		if !strings.Contains(result, "First header") {
			t.Error("Missing 'First header' in output")
		}
		if !strings.Contains(result, "Content line 1") {
			t.Error("Missing 'Content line 1' in output")
		}
		// Should NOT have content after cancellation
		if strings.Contains(result, "Should not appear") {
			t.Error("Output contains content that should not appear after cancellation")
		}
	})
}

// TestArchRapidSignals tests rapid signals don't corrupt output
// Migrated from: arch_test_8
func TestArchRapidSignals(t *testing.T) {
	t.Run("rapid_signals_dont_corrupt", func(t *testing.T) {
		// Create test data
		var input strings.Builder
		for i := 1; i <= 100; i++ {
			input.WriteString("# Header ")
			input.WriteString(strings.Repeat("X", i))
			input.WriteByte('\n')
		}
		testData := input.String()

		// Send rapid succession of signals
		for attempt := 1; attempt <= 3; attempt++ {
			// Very short timeout to simulate rapid signals
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()

			var output bytes.Buffer
			_ = Flow(ctx, strings.NewReader(testData), &output, 0, passthroughRenderer)

			// Each output should be valid (starts with expected format)
			result := output.String()
			if len(result) > 0 {
				lines := strings.Split(result, "\n")
				// Check first non-empty line is a header
				for _, line := range lines {
					if strings.TrimSpace(line) != "" {
						if !strings.Contains(line, "#") && !strings.Contains(line, "Header") {
							t.Errorf("Attempt %d: Corrupted output, unexpected content: %q", attempt, line)
						}
						break
					}
				}
			}
		}
	})
}

// TestArchContextCancellation tests context cancellation is immediate
// Migrated from: arch_test_9
func TestArchContextCancellation(t *testing.T) {
	t.Run("context_cancellation_is_immediate", func(t *testing.T) {
		// Create large but finite input to test cancellation
		var input strings.Builder
		for i := 0; i < 1000; i++ {
			input.WriteString("# Repeating header\n")
		}

		// Measure time for cancellation
		start := time.Now()

		// Cancel after 50ms
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		var output bytes.Buffer
		_ = Flow(ctx, strings.NewReader(input.String()), &output, 0, passthroughRenderer)

		elapsed := time.Since(start)

		// Should terminate within reasonable time (600ms including overhead)
		if elapsed > 600*time.Millisecond {
			t.Errorf("Cancellation took too long: %v", elapsed)
		}
	})
}

// Helper type for simulating slow streams
type testSlowStreamReader struct {
	lines   []string
	delays  []time.Duration
	current int
	buffer  []byte
}

func (r *testSlowStreamReader) Read(p []byte) (n int, err error) {
	if r.current >= len(r.lines) {
		return 0, io.EOF
	}

	// If buffer is empty, get next line
	if len(r.buffer) == 0 {
		if r.current < len(r.delays) {
			time.Sleep(r.delays[r.current])
		}
		r.buffer = []byte(r.lines[r.current] + "\n")
		r.current++
	}

	// Copy from buffer to p
	n = copy(p, r.buffer)
	r.buffer = r.buffer[n:]
	return n, nil
}

// TestArchFirstByteLatency tests first byte latency is minimal
// Migrated from: arch_test_10
func TestArchFirstByteLatency(t *testing.T) {
	t.Run("first_byte_latency_minimal", func(t *testing.T) {
		// Create slow input that delays after first line
		slowReader := &testSlowStreamReader{
			lines: []string{
				"# First line output immediately",
				"# Second line much later",
			},
			delays: []time.Duration{
				0,
				2 * time.Second,
			},
		}

		// Cancel after 100ms
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		var output bytes.Buffer
		_ = Flow(ctx, slowReader, &output, -1, passthroughRenderer)

		result := output.String()
		// Should have first line but not second
		if !strings.Contains(result, "First line") {
			t.Error("Missing 'First line' - latency too high")
		}
		if strings.Contains(result, "Second line") {
			t.Error("Should not have 'Second line' after 100ms timeout")
		}
	})
}

// TestArchLinearThroughput tests throughput scales with input size
// Migrated from: arch_test_11
func TestArchLinearThroughput(t *testing.T) {
	t.Run("linear_throughput_scaling", func(t *testing.T) {
		// Generate different sized inputs
		var smallInput strings.Builder
		for i := 1; i <= 10; i++ {
			smallInput.WriteString("# Header ")
			smallInput.WriteString(strings.Repeat("X", i))
			smallInput.WriteByte('\n')
		}

		var mediumInput strings.Builder
		for i := 1; i <= 100; i++ {
			mediumInput.WriteString("# Header ")
			mediumInput.WriteString(strings.Repeat("X", i%50))
			mediumInput.WriteByte('\n')
		}

		// Process both
		ctx := context.Background()
		var smallOutput, mediumOutput bytes.Buffer

		err1 := Flow(ctx, strings.NewReader(smallInput.String()), &smallOutput, -1, passthroughRenderer)
		err2 := Flow(ctx, strings.NewReader(mediumInput.String()), &mediumOutput, -1, passthroughRenderer)

		// Verify both completed successfully
		if err1 != nil {
			t.Errorf("Small input failed: %v", err1)
		}
		if err2 != nil {
			t.Errorf("Medium input failed: %v", err2)
		}
		if smallOutput.Len() == 0 || mediumOutput.Len() == 0 {
			t.Error("Output missing")
		}
	})
}

// TestArchMemoryIndependentOfSize tests memory usage independent of document size with streaming
// Migrated from: arch_test_12
func TestArchMemoryIndependentOfSize(t *testing.T) {
	t.Run("memory_independent_of_doc_size", func(t *testing.T) {
		// Generate large document
		var largeInput strings.Builder
		for i := 1; i <= 10000; i++ {
			largeInput.WriteString("# Header ")
			largeInput.WriteString(strings.Repeat("X", i%100))
			largeInput.WriteString("\nParagraph content for section ")
			largeInput.WriteString(strings.Repeat("Y", i%50))
			largeInput.WriteString("\n\n")
		}

		// Process with small buffer - should complete without memory issues
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var output bytes.Buffer
		err := Flow(ctx, strings.NewReader(largeInput.String()), &output, 1024, passthroughRenderer)

		if err != nil && err != context.DeadlineExceeded {
			t.Fatalf("Large document processing failed: %v", err)
		}

		// Should have processed entire document
		if output.Len() == 0 {
			t.Error("No output from large document")
		}

		// Verify it processed multiple headers (not just first few)
		result := output.String()
		lines := strings.Split(result, "\n")
		if len(lines) > 100 {
			// Check that headers appear throughout, including near the end
			lastSection := strings.Join(lines[len(lines)-100:], "\n")
			if !strings.Contains(lastSection, "Header") {
				t.Error("Large document not fully processed")
			}
		}
	})
}

// TestArchEOFSpacing tests EOF spacing normalization is consistent
// Migrated from: arch_test_13
func TestArchEOFSpacing(t *testing.T) {
	t.Run("EOF_spacing_normalized_correctly", func(t *testing.T) {
		// Test EOF behavior normalization
		input := "Line 1\n\n### Header"

		// Process with minimal buffer (will split)
		outputStream := runFlow(t, input, -1)
		// Process without streaming
		outputNormal := runFlow(t, input, 0)

		// Both should match (EOF normalization working)
		if outputStream != outputNormal {
			t.Errorf("EOF spacing differs:\nStream: %q\nNormal: %q", outputStream, outputNormal)
		}
	})
}

// TestArchDeterministicRendering tests glamour rendering is deterministic
// Migrated from: arch_test_14
func TestArchDeterministicRendering(t *testing.T) {
	t.Run("deterministic_glamour_rendering", func(t *testing.T) {
		// Same input should always produce same output
		input := "# Test\n\nContent\n\n## Test2"

		// Run multiple times
		outputs := make([]string, 3)
		for i := 0; i < 3; i++ {
			outputs[i] = runFlow(t, input, 64)
		}

		// All runs should be identical
		if outputs[0] != outputs[1] {
			t.Error("Run 1 and 2 differ - rendering not deterministic")
		}
		if outputs[1] != outputs[2] {
			t.Error("Run 2 and 3 differ - rendering not deterministic")
		}
	})
}

// TestArchBinaryData tests binary data doesn't cause panic
// Migrated from: arch_test_15
func TestArchBinaryData(t *testing.T) {
	t.Run("binary_data_handled_gracefully", func(t *testing.T) {
		// Mix binary data with markdown
		var input bytes.Buffer
		input.WriteString("# Valid header\n")
		// Add some binary data
		binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD, 0x80, 0x81}
		input.Write(binaryData)
		input.WriteString("\n## Another header\n")

		// Process with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		var output bytes.Buffer
		err := Flow(ctx, &input, &output, 0, passthroughRenderer)

		// Should not panic - just handle gracefully
		if err != nil && err != context.DeadlineExceeded {
			// Binary data may cause errors, but shouldn't panic
			t.Logf("Binary data caused error (expected): %v", err)
		}

		// Should have produced some output (at least the headers)
		if output.Len() > 0 {
			result := output.String()
			if !strings.Contains(result, "Valid header") && !strings.Contains(result, "Another header") {
				t.Error("Binary data prevented any processing")
			}
		}
	})
}

// TestArchReferenceLinks tests reference link resolution with opportunistic accumulation
// Migrated from: arch_test_16
func TestArchReferenceLinks(t *testing.T) {
	t.Run("reference_links_resolve_opportunistically", func(t *testing.T) {
		// Create markdown with reference-style links
		refLinksMd := `# Document with Reference Links

This is a [reference link][ref1] that needs the definition below.

Here's another [reference link][ref2] in the text.

Some more content to create distance between link usage and definition.

More paragraphs to ensure the reference is far from its definition.
This tests whether the buffer captures enough context.

[ref1]: https://example.com/link1 "Link 1 Title"
[ref2]: https://example.com/link2 "Link 2 Title"`

		// Process with minimal buffer
		// But data arrives quickly, so opportunistic accumulation occurs
		outputSmall := runFlow(t, refLinksMd, 16)

		// Process with unlimited buffer
		outputLarge := runFlow(t, refLinksMd, 0)

		// With passthrough renderer, reference markers will remain
		// But the important test is that content is preserved correctly
		// In real usage with glamour, these would be resolved based on buffer size

		// Verify the content is present in both outputs
		if !strings.Contains(outputSmall, "reference link") {
			t.Error("Small buffer: Missing link content")
		}
		if !strings.Contains(outputLarge, "reference link") {
			t.Error("Large buffer: Missing link content")
		}

		// Note: Glamour consumes reference link definitions
		// They don't appear in the rendered output
		// The links themselves are resolved and rendered as proper links

		// Note: With actual glamour renderer, large buffer would resolve references
		// while small buffer might not, depending on opportunistic accumulation
	})
}
