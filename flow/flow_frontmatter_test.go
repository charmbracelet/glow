package flow

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestFrontmatter1SimpleFrontmatter(t *testing.T) {
	t.Parallel()

	content := `---
title: Test Document
author: Test Author
---

# Main Content

This is the main content.`

	// Compare flow=-1 vs flow=0
	output1 := runFlow(t, content, -1)
	output2 := runFlow(t, content, 0)
	compareOutputs(t, []byte(output1), []byte(output2))
}

func TestFrontmatter2SplitAcrossChunks(t *testing.T) {
	t.Parallel()

	content := `---
title: This is a very long title that will definitely span across chunk boundaries
author: Author Name
date: 2024-01-01
tags: [test, streaming, markdown]
---

# Content After Frontmatter

Regular content here.`

	// Compare flow=0 vs flow=32
	output1 := runFlow(t, content, 0)
	output2 := runFlow(t, content, 32)
	compareOutputs(t, []byte(output1), []byte(output2))
}

func TestFrontmatter3LargeFrontmatter(t *testing.T) {
	t.Parallel()

	// Generate large frontmatter (2KB)
	var buf bytes.Buffer
	buf.WriteString("---\n")
	for i := 1; i <= 50; i++ {
		fmt.Fprintf(&buf, "field_%d: This is a long value for field number %d to create bulk\n", i, i)
	}
	buf.WriteString("---\n\n")
	buf.WriteString("# Content\n")
	buf.WriteString("After large frontmatter\n")

	content := buf.String()

	// Compare flow=0 vs flow=512
	output1 := runFlow(t, content, 0)
	output2 := runFlow(t, content, 512)
	compareOutputs(t, []byte(output1), []byte(output2))
}

func TestFrontmatter4MalformedFrontmatter(t *testing.T) {
	t.Parallel()

	content := `---
title: Unclosed Frontmatter
author: Test

# This looks like content but frontmatter never closed

More content here.`

	// Should handle gracefully - glow will process this as content since frontmatter never closes
	output := runFlow(t, content, 64)

	// Verify some output was produced (should treat entire doc as content)
	if len(output) == 0 {
		t.Errorf("No output produced for malformed frontmatter")
	}
	t.Logf("Malformed frontmatter handling - output length: %d bytes", len(output))
}

func TestFrontmatter5EmptyFrontmatter(t *testing.T) {
	t.Parallel()

	content := `---
---

# Main Content

Document with empty frontmatter.`

	// Compare flow=0 vs flow=32
	output1 := runFlow(t, content, 0)
	output2 := runFlow(t, content, 32)
	compareOutputs(t, []byte(output1), []byte(output2))
}

func TestFrontmatter6NoFrontmatter(t *testing.T) {
	t.Parallel()

	content := `# Document Without Frontmatter

Just regular markdown content here.
No frontmatter at all.`

	// Compare flow=0 vs flow=32
	output1 := runFlow(t, content, 0)
	output2 := runFlow(t, content, 32)
	compareOutputs(t, []byte(output1), []byte(output2))
}

func TestFrontmatter7DashesInContent(t *testing.T) {
	t.Parallel()

	content := `---
title: Test
---

# Content

Some text here.

---

More content after horizontal rule.`

	// Compare flow=0 vs flow=64
	output1 := runFlow(t, content, 0)
	output2 := runFlow(t, content, 64)
	compareOutputs(t, []byte(output1), []byte(output2))
}

func TestFrontmatter8StreamingViaStdin(t *testing.T) {
	t.Parallel()

	content := `---
title: Stdin Test
---

# Streamed Content

This comes from stdin.`

	// Compare flow=0 vs flow=32
	output1 := runFlow(t, content, 0)
	output2 := runFlow(t, content, 32)
	compareOutputs(t, []byte(output1), []byte(output2))
}

func TestFrontmatter9BoundaryAtFlowSize(t *testing.T) {
	t.Parallel()

	content := `---
title: X
desc: YZ
---

# Content Here

Body text.`

	// Calculate exact frontmatter size
	lines := strings.Split(content, "\n")
	frontmatterEnd := 0
	for i, line := range lines {
		if i > 0 && line == "---" {
			frontmatterEnd = i
			break
		}
	}

	// Get frontmatter portion including trailing newline
	frontmatterLines := lines[:frontmatterEnd+1]
	frontmatter := strings.Join(frontmatterLines, "\n") + "\n"
	fmSize := int64(len(frontmatter))

	t.Logf("Frontmatter size: %d bytes", fmSize)

	// Compare flow=0 vs flow=fmSize
	output1 := runFlow(t, content, 0)
	output2 := runFlow(t, content, fmSize)
	compareOutputs(t, []byte(output1), []byte(output2))
}

func TestFrontmatter10CodeBlockMarkers(t *testing.T) {
	t.Parallel()

	content := `---
title: Code Example
---

# Example

` + "```yaml" + `
---
config: value
---
` + "```" + `

More content.`

	// Compare flow=0 vs flow=32
	output1 := runFlow(t, content, 0)
	output2 := runFlow(t, content, 32)
	compareOutputs(t, []byte(output1), []byte(output2))
}
