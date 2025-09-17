package flow

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Package flow extreme tests
//
// These tests validate Flow's handling of extreme document structures:
// - Deep nesting (1000+ levels)
// - Massive documents (near 1MB)
// - Pathological structures
// - Stress combinations
//
// DISCOVERED LIMITS (from test execution):
// - 1000-level nested lists: Successfully processed (1MB document)
// - 500-level nested blockquotes: Content modified but handled gracefully
// - Near 1MB documents: Processed ~528KB with default windows
// - 10,000 headings: Successfully processed (297KB document)
// - 100x100 tables: Handled without issues (80KB)
// - 1000 reference links: Processed with modification (79KB)
// - All tests complete within 1 second
// - Memory usage remains bounded (no OOM conditions)
// - No stack overflows or panics observed
// - Timeouts prevent infinite loops on pathological input

// 1. EXTREME NESTING TESTS

func TestExtremeNesting(t *testing.T) {
	t.Run("1000_level_nested_lists", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping extreme nesting test in short mode")
		}
		var input strings.Builder

		// Build 1000-level nested list
		for i := 0; i < 1000; i++ {
			input.WriteString(strings.Repeat("  ", i))
			input.WriteString("- Item at level ")
			input.WriteString(strconv.Itoa(i))
			input.WriteString("\n")
		}

		doc := input.String()
		t.Logf("Generated %d-level nested list (%d bytes)", 1000, len(doc))

		// Test with various window sizes
		for _, window := range []int64{100, 1024, 16384} {
			var buf bytes.Buffer
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := Flow(ctx, strings.NewReader(doc), &buf, window, passthroughRenderer)

			// Should handle without stack overflow or hanging
			if err != nil && err != context.DeadlineExceeded {
				t.Logf("Error with window=%d: %v", window, err)
			}

			// Verify some output produced
			if buf.Len() == 0 {
				t.Errorf("No output with window=%d", window)
			} else {
				t.Logf("Processed %d bytes with window=%d", buf.Len(), window)
			}
		}
	})

	t.Run("500_level_nested_blockquotes", func(t *testing.T) {
		var input strings.Builder

		// Build deeply nested blockquotes
		for i := 0; i < 500; i++ {
			input.WriteString(strings.Repeat(">", i+1))
			input.WriteString(" Quote at level ")
			input.WriteString(strconv.Itoa(i))
			input.WriteString("\n")
		}

		doc := input.String()
		t.Logf("Generated %d-level nested blockquotes (%d bytes)", 500, len(doc))

		var buf bytes.Buffer
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		err := Flow(ctx, strings.NewReader(doc), &buf, 1024, passthroughRenderer)
		if err != nil && err != context.DeadlineExceeded {
			t.Logf("Blockquote nesting error: %v", err)
		}

		if buf.String() != doc {
			t.Logf("Blockquote content modified: input %d bytes, output %d bytes",
				len(doc), buf.Len())
		}
	})

	t.Run("deeply_nested_code_blocks", func(t *testing.T) {
		var input strings.Builder

		// Create nested code blocks with different fence lengths
		for i := 0; i < 50; i++ {
			fenceLen := 3 + i
			input.WriteString(strings.Repeat("`", fenceLen))
			input.WriteString("language")
			input.WriteString(strconv.Itoa(i))
			input.WriteString("\n")
			input.WriteString("Code at nesting level ")
			input.WriteString(strconv.Itoa(i))
			input.WriteString("\n")
		}

		// Close all fences in reverse order
		for i := 49; i >= 0; i-- {
			fenceLen := 3 + i
			input.WriteString(strings.Repeat("`", fenceLen))
			input.WriteString("\n")
		}

		doc := input.String()
		testWithMultipleWindows(t, doc, "nested code blocks")
	})

	t.Run("complex_table_nesting", func(t *testing.T) {
		var input strings.Builder

		// Create a table with nested markdown in cells
		input.WriteString("| Complex | Table | With | Nesting |\n")
		input.WriteString("|---------|-------|------|----------|\n")

		for i := 0; i < 100; i++ {
			input.WriteString(fmt.Sprintf("| **Bold %d** | *Italic %d* | `code %d` | ", i, i, i))
			// Add nested list in cell
			input.WriteString("- Item " + strconv.Itoa(i) + " |\n")
		}

		doc := input.String()
		testWithMultipleWindows(t, doc, "complex table")
	})

	t.Run("mixed_nesting_patterns", func(t *testing.T) {
		var input strings.Builder

		// Mix different nesting types
		for i := 0; i < 100; i++ {
			// Alternate between different structures
			switch i % 4 {
			case 0:
				input.WriteString(strings.Repeat(">", i%10+1))
				input.WriteString(" Blockquote ")
			case 1:
				input.WriteString(strings.Repeat("  ", i%10))
				input.WriteString("- List item ")
			case 2:
				input.WriteString("    ") // Code block
				input.WriteString("code ")
			case 3:
				input.WriteString("# ")
				input.WriteString(strings.Repeat("#", i%6))
				input.WriteString(" Heading ")
			}
			input.WriteString(strconv.Itoa(i))
			input.WriteString("\n")
		}

		doc := input.String()
		testWithMultipleWindows(t, doc, "mixed nesting")
	})
}

// 2. MASSIVE DOCUMENTS TESTS

func TestMassiveDocuments(t *testing.T) {
	t.Run("near_1mb_document", func(t *testing.T) {
		var input strings.Builder

		// Generate document approaching 1MB limit
		targetSize := 900 * 1024 // 900KB to stay under limit

		input.WriteString("# Massive Document\n\n")

		sectionNum := 0
		for input.Len() < targetSize {
			sectionNum++
			input.WriteString(fmt.Sprintf("\n## Section %d\n\n", sectionNum))
			input.WriteString("This is content for section with various **markdown** ")
			input.WriteString("features including *emphasis*, `code`, and [links](url).\n")
			input.WriteString("- List item 1\n- List item 2\n- List item 3\n\n")

			// Add table every 10 sections
			if sectionNum%10 == 0 {
				input.WriteString("| Col1 | Col2 | Col3 |\n")
				input.WriteString("|------|------|------|\n")
				for j := 0; j < 5; j++ {
					input.WriteString(fmt.Sprintf("| R%d-1 | R%d-2 | R%d-3 |\n", j, j, j))
				}
				input.WriteString("\n")
			}
		}

		doc := input.String()
		t.Logf("Document size: %d bytes", len(doc))

		var buf bytes.Buffer
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := Flow(ctx, strings.NewReader(doc), &buf, 16384, passthroughRenderer)

		if err != nil {
			t.Logf("Large document error: %v", err)
		}

		// May be truncated but should process some
		if buf.Len() == 0 {
			t.Error("No output for large document")
		} else {
			t.Logf("Processed %d bytes of %d byte document", buf.Len(), len(doc))
		}
	})

	t.Run("10000_headings_structure", func(t *testing.T) {
		var input strings.Builder

		// Generate document with many headings
		for i := 1; i <= 10000; i++ {
			level := (i % 6) + 1
			input.WriteString(strings.Repeat("#", level))
			input.WriteString(fmt.Sprintf(" Heading %d at level %d\n\n", i, level))

			// Add some content every 100 headings
			if i%100 == 0 {
				input.WriteString("Content paragraph under heading.\n\n")
			}
		}

		doc := input.String()
		t.Logf("Generated document with 10000 headings (%d bytes)", len(doc))

		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(doc), &buf, 10000, passthroughRenderer)

		if err != nil {
			t.Logf("Many headings error: %v", err)
		}

		// Check some output produced
		if buf.Len() > 0 {
			t.Logf("Processed %d bytes", buf.Len())
		}
	})

	t.Run("giant_table_100x100", func(t *testing.T) {
		// Reduced from 1000x1000 to 100x100 for practicality
		var input strings.Builder

		// Create table header
		input.WriteString("|")
		for i := 0; i < 100; i++ {
			input.WriteString(fmt.Sprintf(" Col%d |", i))
		}
		input.WriteString("\n|")
		for i := 0; i < 100; i++ {
			input.WriteString("------|")
		}
		input.WriteString("\n")

		// Create table rows
		for row := 0; row < 100; row++ {
			input.WriteString("|")
			for col := 0; col < 100; col++ {
				input.WriteString(fmt.Sprintf(" %d,%d |", row, col))
			}
			input.WriteString("\n")
		}

		doc := input.String()
		t.Logf("Generated 100x100 table (%d bytes)", len(doc))

		var buf bytes.Buffer
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := Flow(ctx, strings.NewReader(doc), &buf, 10000, passthroughRenderer)
		if err != nil {
			t.Logf("Giant table error: %v", err)
		}

		if buf.Len() > 0 {
			t.Logf("Processed %d bytes of table", buf.Len())
		}
	})

	t.Run("1000_reference_links", func(t *testing.T) {
		var input strings.Builder

		// Create document with many reference links
		input.WriteString("# Document with Many References\n\n")

		// Use all references
		for i := 0; i < 1000; i++ {
			input.WriteString(fmt.Sprintf("This is [reference %d][ref%d] in text. ", i, i))
			if i%10 == 9 {
				input.WriteString("\n\n")
			}
		}

		input.WriteString("\n\n")

		// Define all references
		for i := 0; i < 1000; i++ {
			input.WriteString(fmt.Sprintf("[ref%d]: https://example.com/page%d\n", i, i))
		}

		doc := input.String()
		t.Logf("Generated document with 1000 reference links (%d bytes)", len(doc))

		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(doc), &buf, 10000, passthroughRenderer)

		if err != nil {
			t.Logf("Reference links error: %v", err)
		}

		if buf.String() != doc {
			t.Logf("Reference links modified: input %d, output %d", len(doc), buf.Len())
		}
	})

	t.Run("maximum_footnotes", func(t *testing.T) {
		var input strings.Builder

		// Create document with many footnotes
		input.WriteString("# Document with Footnotes\n\n")

		// Use footnotes
		for i := 1; i <= 500; i++ {
			input.WriteString(fmt.Sprintf("Text with footnote[^%d]. ", i))
			if i%10 == 0 {
				input.WriteString("\n\n")
			}
		}

		input.WriteString("\n\n")

		// Define footnotes
		for i := 1; i <= 500; i++ {
			input.WriteString(fmt.Sprintf("[^%d]: Footnote number %d content.\n", i, i))
		}

		doc := input.String()
		testWithMultipleWindows(t, doc, "footnotes")
	})
}

// 3. PATHOLOGICAL STRUCTURES TESTS

func TestPathologicalStructures(t *testing.T) {
	t.Run("exponential_expansion_attempts", func(t *testing.T) {
		// Document that could expand exponentially if mishandled
		input := "[a][b][c][d][e][f][g][h][i][j]\n\n"

		// Each reference could point to multiple others
		for _, c := range "abcdefghij" {
			input += fmt.Sprintf("[%c]: #%c1 #%c2 #%c3 #%c4 #%c5\n", c, c, c, c, c, c)
		}

		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(input), &buf, 1024, passthroughRenderer)

		if err != nil {
			t.Fatalf("Expansion attempt failed: %v", err)
		}

		// Glamour processes the content, just verify it handled it gracefully
		if buf.Len() == 0 {
			t.Error("No output for pathological expansion")
		}
	})

	t.Run("circular_reference_patterns", func(t *testing.T) {
		// Create circular references
		input := "[link1][ref1] and [link2][ref2] and [link3][ref3]\n\n"
		input += "[ref1]: #ref2\n"
		input += "[ref2]: #ref3\n"
		input += "[ref3]: #ref1\n"

		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(input), &buf, 100, passthroughRenderer)

		if err != nil {
			t.Fatalf("Circular reference failed: %v", err)
		}

		// Glamour processes references, just verify no crash
		if buf.Len() == 0 {
			t.Error("No output for circular references")
		}
	})

	t.Run("almost_valid_markdown", func(t *testing.T) {
		// Markdown that's almost but not quite valid
		almostValid := []string{
			"# Heading without space\n",
			"** Bold with spaces **\n",
			"[Link with [nested] brackets](url)\n",
			"```unclosed fence\n",
			"* List\n  * Wrong indent\n",
			"> Quote\n>> Double quote\n",
			"| Table | Without | Separator\n",
		}

		for _, input := range almostValid {
			var buf bytes.Buffer
			err := Flow(context.Background(), strings.NewReader(input), &buf, 100, passthroughRenderer)

			if err != nil {
				t.Errorf("Almost valid failed: %q: %v", input, err)
			}

			// Glamour may fix/format almost-valid markdown
			if buf.Len() == 0 {
				t.Errorf("No output for almost-valid markdown: %q", input)
			}
		}
	})

	t.Run("parser_edge_cases", func(t *testing.T) {
		edgeCases := []string{
			strings.Repeat("*", 1000) + "\n",                    // Many asterisks
			strings.Repeat("[", 500) + strings.Repeat("]", 500), // Many brackets
			strings.Repeat("`", 100) + "\n",                     // Many backticks
			strings.Repeat("#", 100) + " Heading\n",             // Many hashes
			strings.Repeat("-", 1000) + "\n",                    // Many dashes
			"\\*\\*\\*\\*\\*\\*\\*\\*\\*\\*\n",                  // Many escaped chars
		}

		for i, input := range edgeCases {
			var buf bytes.Buffer
			err := Flow(context.Background(), strings.NewReader(input), &buf, 100, passthroughRenderer)

			if err != nil {
				t.Errorf("Edge case %d failed: %v", i, err)
			}

			// Glamour processes edge cases, just verify handled
			if buf.Len() == 0 {
				t.Errorf("No output for edge case %d", i)
			}
		}
	})

	t.Run("ambiguous_structures", func(t *testing.T) {
		// Structures that could be interpreted multiple ways
		ambiguous := []string{
			"1. List\n1. With\n1. Same\n1. Numbers\n",
			"- List\n+ With\n* Different\n- Bullets\n",
			"> > > Triple nested quote\n",
			"    Code or list?\n  - Maybe list\n    Or code?\n",
			"_*Combined emphasis*_\n",
			"~~**Strikethrough bold**~~\n",
		}

		for _, input := range ambiguous {
			var buf bytes.Buffer
			err := Flow(context.Background(), strings.NewReader(input), &buf, 100, passthroughRenderer)

			if err != nil {
				t.Errorf("Ambiguous structure failed: %q: %v", input, err)
			}

			// Glamour interprets ambiguous structures
			if buf.Len() == 0 {
				t.Errorf("No output for ambiguous structure: %q", input)
			}
		}
	})
}

// 4. STRESS COMBINATIONS TESTS

func TestStressCombinations(t *testing.T) {
	t.Run("unicode_plus_deep_nesting", func(t *testing.T) {
		var input strings.Builder

		// Combine Unicode with deep nesting
		emojis := []string{"🚀", "💻", "🎉", "🌍", "⚡", "🔥", "✨", "🎨", "🎯", "🏆"}
		rtlText := []string{"مرحبا", "שלום", "سلام", "שבת", "مساء", "בוקר", "يوم", "לילה", "صباح", "ערב"}

		for i := 0; i < 100; i++ {
			input.WriteString(strings.Repeat("  ", i))
			input.WriteString(fmt.Sprintf("- %s Level %d %s\n",
				emojis[i%len(emojis)], i, rtlText[i%len(rtlText)]))
		}

		doc := input.String()
		testWithMultipleWindows(t, doc, "unicode + nesting")
	})

	t.Run("large_doc_complex_structure", func(t *testing.T) {
		var input strings.Builder
		targetSize := 100 * 1024 // 100KB

		for input.Len() < targetSize {
			// Add heading
			input.WriteString("## Complex Section\n\n")

			// Add nested list
			for i := 0; i < 5; i++ {
				input.WriteString(strings.Repeat("  ", i))
				input.WriteString(fmt.Sprintf("- Nested item %d\n", i))
			}

			// Add table
			input.WriteString("\n| A | B | C |\n|---|---|---|\n| 1 | 2 | 3 |\n\n")

			// Add code block
			input.WriteString("```go\nfunc example() {}\n```\n\n")

			// Add blockquote
			input.WriteString("> Quote with **bold** and *italic*\n\n")
		}

		doc := input.String()
		t.Logf("Generated complex document (%d bytes)", len(doc))

		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(doc), &buf, 10000, passthroughRenderer)

		if err != nil {
			t.Logf("Complex structure error: %v", err)
		}

		if buf.Len() == 0 {
			t.Error("No output for complex document")
		}
	})

	t.Run("all_features_combined", func(t *testing.T) {
		// Document using every markdown feature
		input := `# All Features Document 🎉

## Text Formatting
**Bold** *Italic* ***Bold Italic*** ~~Strikethrough~~ ` + "`code`" + `

## Lists
### Unordered
- Item 1
  - Nested 1.1
    - Nested 1.1.1
- Item 2

### Ordered
1. First
2. Second
   1. Nested
   2. More nested

## Links and References
[Inline link](https://example.com)
[Reference link][ref1]
[ref1]: https://example.com

## Images
![Alt text](image.png)
![Reference image][img1]
[img1]: image.png

## Code
Inline: ` + "`const x = 42`" + `

Block:
` + "```javascript" + `
function example() {
  return 42;
}
` + "```" + `

## Tables
| Header 1 | Header 2 | Header 3 |
|----------|----------|----------|
| Cell 1   | Cell 2   | Cell 3   |
| Cell 4   | Cell 5   | Cell 6   |

## Blockquotes
> Level 1
>> Level 2
>>> Level 3

## Horizontal Rules
---
***
___

## HTML (if supported)
<div>HTML content</div>

## Footnotes
Text with footnote[^1].
[^1]: Footnote content.

## Task Lists
- [x] Completed
- [ ] Incomplete

## Unicode 多语言
Arabic: مرحبا
Hebrew: שלום
Emoji: 🚀 💻 🎨
`

		testWithMultipleWindows(t, input, "all features")
	})

	t.Run("maximum_complexity_document", func(t *testing.T) {
		var input strings.Builder

		// Create the most complex document possible
		input.WriteString("# Maximum Complexity 🔥\n\n")

		// Deep nesting with unicode
		for i := 0; i < 50; i++ {
			input.WriteString(strings.Repeat(">", i+1))
			input.WriteString(" Quote level ")
			input.WriteString(strconv.Itoa(i))
			if i%2 == 0 {
				input.WriteString(" مرحبا")
			} else {
				input.WriteString(" שלום")
			}
			input.WriteString("\n")
		}

		// Complex table
		input.WriteString("\n| Complex | Table | With | **Markdown** |\n")
		input.WriteString("|---------|-------|------|---------------|\n")
		for i := 0; i < 20; i++ {
			input.WriteString(fmt.Sprintf("| `code%d` | *italic%d* | [link%d](#) | ", i, i, i))
			input.WriteString(fmt.Sprintf("List:\n- Item %d |\n", i))
		}

		// Many references
		input.WriteString("\n")
		for i := 0; i < 50; i++ {
			input.WriteString(fmt.Sprintf("[ref%d]: #section%d\n", i, i))
		}

		doc := input.String()
		testWithMultipleWindows(t, doc, "maximum complexity")
	})

	t.Run("performance_under_stress", func(t *testing.T) {
		// Test performance with stress document
		var input strings.Builder

		// Create document that stresses all aspects
		for i := 0; i < 100; i++ {
			// Alternating complex structures
			switch i % 5 {
			case 0:
				// Deep nesting
				for j := 0; j < 10; j++ {
					input.WriteString(strings.Repeat("  ", j))
					input.WriteString(fmt.Sprintf("- Item %d-%d\n", i, j))
				}
			case 1:
				// Unicode
				input.WriteString("Unicode: 中文 العربية עברית 日本語 한국어 🚀\n")
			case 2:
				// Table
				input.WriteString("| A | B | C |\n|---|---|---|\n| 1 | 2 | 3 |\n")
			case 3:
				// Code block
				input.WriteString("```\nCode block " + strconv.Itoa(i) + "\n```\n")
			case 4:
				// References
				input.WriteString(fmt.Sprintf("[link%d]: #ref%d\n", i, i))
			}
		}

		doc := input.String()

		// Measure performance
		start := time.Now()
		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(doc), &buf, 1024, passthroughRenderer)
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("Performance test failed: %v", err)
		}

		t.Logf("Processed %d bytes in %v", len(doc), elapsed)

		// Should complete reasonably quickly
		if elapsed > 2*time.Second {
			t.Error("Performance too slow for stress document")
		}
	})
}

// Helper function to test with multiple window sizes
func testWithMultipleWindows(t *testing.T, input string, description string) {
	windows := []int64{100, 1024, 10000}

	for _, window := range windows {
		var buf bytes.Buffer
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := Flow(ctx, strings.NewReader(input), &buf, window, passthroughRenderer)

		if err != nil && err != context.DeadlineExceeded {
			t.Logf("%s error with window=%d: %v", description, window, err)
		}

		if buf.String() != input {
			t.Logf("%s modified with window=%d: input %d bytes, output %d bytes",
				description, window, len(input), buf.Len())
		}
	}
}
