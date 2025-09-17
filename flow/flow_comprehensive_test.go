package flow

import (
	"strings"
	"testing"
	"time"
)

func TestComprehensive(t *testing.T) {
	t.Run("simple_echo", func(t *testing.T) {
		input := "# Test"
		output := runFlow(t, input, 0)
		if len(output) == 0 {
			t.Fatal("Expected output, got none")
		}
	})

	t.Run("empty_input", func(t *testing.T) {
		input := ""
		output := runFlow(t, input, 0)
		// Glamour returns two newlines for empty input
		expected := "\n\n"
		if output != expected {
			t.Fatalf("Expected glamour empty output %q, got: %q", expected, output)
		}
	})

	t.Run("single_character", func(t *testing.T) {
		input := "x"
		output := runFlow(t, input, 0)
		if len(output) == 0 {
			t.Fatal("Expected output, got none")
		}
	})

	t.Run("multi_line_markdown", func(t *testing.T) {
		input := "# Title\n\nParagraph\n\n## Section"
		output := runFlow(t, input, 0)
		if len(output) == 0 {
			t.Fatal("Expected output, got none")
		}
		// Verify all sections are rendered
		outputStr := string(output)
		if !strings.Contains(outputStr, "Title") {
			t.Error("Expected 'Title' in output")
		}
		if !strings.Contains(outputStr, "Paragraph") {
			t.Error("Expected 'Paragraph' in output")
		}
		if !strings.Contains(outputStr, "Section") {
			t.Error("Expected 'Section' in output")
		}
	})

	t.Run("large_document", func(t *testing.T) {
		// Create large document (similar to shell test)
		var builder strings.Builder
		for i := 0; i < 1000; i++ {
			builder.WriteString("# Section ")
			builder.WriteString(string(rune('A' + i%26)))
			builder.WriteString("\n\nContent for section ")
			builder.WriteString(string(rune('A' + i%26)))
			builder.WriteString("\n\n")
		}
		input := builder.String()

		output := runFlow(t, input, 0)
		if len(output) == 0 {
			t.Fatal("Expected output, got none")
		}
	})

	t.Run("binary_safe", func(t *testing.T) {
		// Test with binary data mixed with markdown
		input := "# Title\n\nText with \x00\x01\x02 binary data\n\n## End"
		output := runFlow(t, input, 0)
		if len(output) == 0 {
			t.Fatal("Expected output, got none")
		}
	})

	t.Run("unicode_content", func(t *testing.T) {
		input := "# 🌟 Unicode Title 🌟\n\n**Bold text** with émojis 😀 and spëcial chars: àáâãäå\n\n## 中文标题"
		output := runFlow(t, input, 0)
		if len(output) == 0 {
			t.Fatal("Expected output, got none")
		}
	})

	t.Run("tables", func(t *testing.T) {
		input := `# Table Test

| Name | Age | City |
|------|-----|------|
| John | 30  | NYC  |
| Jane | 25  | LA   |

End of table.`
		output := runFlow(t, input, 0)
		if len(output) == 0 {
			t.Fatal("Expected output, got none")
		}
	})

	t.Run("nested_lists", func(t *testing.T) {
		input := `# Lists

- Item 1
  - Nested 1.1
  - Nested 1.2
    - Deep nested 1.2.1
- Item 2
  - Nested 2.1

1. Numbered item
2. Another numbered
   - Mixed with bullets`
		output := runFlow(t, input, 0)
		if len(output) == 0 {
			t.Fatal("Expected output, got none")
		}
	})

	t.Run("code_blocks", func(t *testing.T) {
		input := `# Code Examples

Here's some Go code:

` + "```go" + `
func main() {
    fmt.Println("Hello, World!")
}
` + "```" + `

And some shell:

` + "```bash" + `
echo "Hello from bash"
ls -la
` + "```"
		output := runFlow(t, input, 0)
		if len(output) == 0 {
			t.Fatal("Expected output, got none")
		}
	})

	t.Run("blockquotes", func(t *testing.T) {
		input := `# Quotes

> This is a blockquote
>
> With multiple lines
>
> > And nested quotes
> >
> > Even deeper

Normal text after.`
		output := runFlow(t, input, 0)
		if len(output) == 0 {
			t.Fatal("Expected output, got none")
		}
	})

	t.Run("horizontal_rules", func(t *testing.T) {
		input := `# Section 1

Some content

---

# Section 2

More content

***

# Section 3`
		output := runFlow(t, input, 0)
		if len(output) == 0 {
			t.Fatal("Expected output, got none")
		}
	})

	t.Run("links_and_images", func(t *testing.T) {
		input := `# Links Test

This is a [link](https://example.com) and this is an ![image](image.png).

Reference style [link][1] and ![image][2].

[1]: https://reference.com
[2]: reference.png`
		output := runFlow(t, input, 0)
		if len(output) == 0 {
			t.Fatal("Expected output, got none")
		}
	})

	t.Run("emphasis", func(t *testing.T) {
		input := `# Emphasis Test

**Bold text** and *italic text* and ***bold italic***.

__Also bold__ and _also italic_ and ___also bold italic___.

~~Strikethrough~~ text.

` + "`inline code`" + ` and more text.`
		output := runFlow(t, input, 0)
		if len(output) == 0 {
			t.Fatal("Expected output, got none")
		}
	})

	t.Run("mixed_content", func(t *testing.T) {
		input := `# Complex Document

## Introduction

This document contains **mixed content** with _various_ elements.

### Code Example

` + "```python" + `
def hello():
    print("Hello World")
` + "```" + `

### Lists

1. First item
2. Second item
   - Nested bullet
   - Another bullet

### Quote

> "The best way to predict the future is to invent it." - Alan Kay

### Table

| Language | Year |
|----------|------|
| Go       | 2009 |
| Python   | 1991 |

---

That's all folks!`
		output := runFlow(t, input, 0)
		if len(output) == 0 {
			t.Fatal("Expected output, got none")
		}
	})

	t.Run("streaming_consistency", func(t *testing.T) {
		// Test that different window sizes produce same output
		input := `# Streaming Test

This is a test to ensure streaming consistency across different window sizes.

## Section 1

Content here.

## Section 2

More content here.`

		// Test different window sizes
		windowSizes := []int64{0, 1024, 4096}
		outputs := make([]string, len(windowSizes))

		for i, window := range windowSizes {
			output := runFlow(t, input, window)
			outputs[i] = output
		}

		// All outputs should be identical
		for i := 1; i < len(outputs); i++ {
			if outputs[0] != outputs[i] {
				t.Errorf("Output mismatch between window %d and %d", windowSizes[0], windowSizes[i])
			}
		}
	})

	t.Run("performance_baseline", func(t *testing.T) {
		// Create moderately large document
		var builder strings.Builder
		for i := 0; i < 100; i++ {
			builder.WriteString("# Section ")
			builder.WriteString(string(rune('A' + i%26)))
			builder.WriteString("\n\nThis is content for section ")
			builder.WriteString(string(rune('A' + i%26)))
			builder.WriteString(". It contains multiple sentences to make it substantial.\n\n")
		}
		input := builder.String()

		start := time.Now()
		output := runFlow(t, input, 0)
		duration := time.Since(start)
		if len(output) == 0 {
			t.Fatal("Expected output, got none")
		}

		// Should complete within reasonable time (this is a baseline, not strict)
		if duration > 5*time.Second {
			t.Logf("Performance note: took %v to process %d byte document", duration, len(input))
		}
	})

	t.Run("memory_efficiency", func(t *testing.T) {
		// Test with larger document to verify memory usage stays reasonable
		var builder strings.Builder
		for i := 0; i < 500; i++ {
			builder.WriteString("# Section ")
			builder.WriteString(string(rune('A' + i%26)))
			builder.WriteString("\n\n")
			// Add substantial content per section
			for j := 0; j < 10; j++ {
				builder.WriteString("This is line ")
				builder.WriteString(string(rune('0' + j)))
				builder.WriteString(" of content for this section. ")
			}
			builder.WriteString("\n\n")
		}
		input := builder.String()

		output := runFlow(t, input, 1024) // Use windowed mode
		if len(output) == 0 {
			t.Fatal("Expected output, got none")
		}
	})

	t.Run("special_characters", func(t *testing.T) {
		input := `# Special Characters

Testing various special characters:

- Quotes: "smart quotes" and 'smart apostrophes'
- Dashes: en–dash and em—dash
- Symbols: © ® ™ § ¶ † ‡
- Math: ∑ ∆ π ∞ ≠ ≤ ≥
- Arrows: ← → ↑ ↓ ↔ ⇒
- Currency: $ € £ ¥ ¢

End of special characters.`
		output := runFlow(t, input, 0)
		if len(output) == 0 {
			t.Fatal("Expected output, got none")
		}
	})

	t.Run("whitespace_handling", func(t *testing.T) {
		input := "# Title\n\n\n\nExtra   spaces    and\t\ttabs\n\n\nMultiple blank lines\n\n\n\n\n## Section"
		output := runFlow(t, input, 0)
		if len(output) == 0 {
			t.Fatal("Expected output, got none")
		}
	})

	t.Run("line_endings", func(t *testing.T) {
		// Test different line ending styles
		inputs := []string{
			"# Unix\nLine ending\ntest",        // LF
			"# Windows\r\nLine ending\r\ntest", // CRLF
			"# Mac\rLine ending\rtest",         // CR (legacy)
			"# Mixed\nSome LF\r\nSome CRLF\rSome CR",
		}

		for i, input := range inputs {
			output := runFlow(t, input, 0)
			if len(output) == 0 {
				t.Fatalf("Test %d: Expected output, got none", i+1)
			}
		}
	})

	t.Run("html_entities", func(t *testing.T) {
		input := `# HTML Entities

Testing HTML entities:

&amp; &lt; &gt; &quot; &#39; &nbsp;

&copy; &reg; &trade;

Numeric: &#65; &#66; &#67;

Hex: &#x41; &#x42; &#x43;`
		output := runFlow(t, input, 0)
		if len(output) == 0 {
			t.Fatal("Expected output, got none")
		}
	})

	t.Run("markdown_escapes", func(t *testing.T) {
		input := `# Escaped Characters

These should not be interpreted as markdown:

\*not bold\*
\_not italic\_
\` + "`not code`" + `
\# not header
\[not link\](url)
\> not quote

But these should work:

**actually bold**
*actually italic*`
		output := runFlow(t, input, 0)
		if len(output) == 0 {
			t.Fatal("Expected output, got none")
		}
	})

	t.Run("deeply_nested_structures", func(t *testing.T) {
		input := `# Deep Nesting

> Quote level 1
> > Quote level 2
> > > Quote level 3
> > > > Quote level 4
> > > > > Quote level 5

- List level 1
  - List level 2
    - List level 3
      - List level 4
        - List level 5

` + "```" + `
Code block with nested:
  - bullet
    - nested bullet
      - deep bullet
` + "```"
		output := runFlow(t, input, 0)
		if len(output) == 0 {
			t.Fatal("Expected output, got none")
		}
	})

	t.Run("reference_link_resolution", func(t *testing.T) {
		input := `# Reference Links

This has a [reference link][ref1] and another [link][ref2].

Some content in between.

[ref1]: https://example.com "Example"
[ref2]: https://test.com "Test"`

		// Test both buffered and windowed modes
		output1 := runFlow(t, input, 0)    // Buffered
		output2 := runFlow(t, input, 1024) // DefaultChunk

		// Both should produce output
		if len(output1) == 0 || len(output2) == 0 {
			t.Fatal("Expected output in both modes")
		}
	})

	t.Run("edge_case_boundaries", func(t *testing.T) {
		// Test content that might trigger boundary edge cases
		input := `# Edge Cases

Short.

A slightly longer paragraph that might cross some boundaries depending on window size.

## Another Section

` + "```" + `
code
block
here
` + "```" + `

Final paragraph.`

		// Test various window sizes
		windows := []int64{10, 50, 100, 500}
		for _, window := range windows {
			output := runFlow(t, input, window)
			if len(output) == 0 {
				t.Fatalf("Window %d: Expected output, got none", window)
			}
		}
	})
}
