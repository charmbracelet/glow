package mermaid

import (
	"strings"
	"testing"
)

func TestProcess_Disabled(t *testing.T) {
	input := "```mermaid\ngraph LR\n    A --> B\n```"
	result := Process(input, false)

	if result != input {
		t.Errorf("disabled processing should return input unchanged\ngot: %s\nwant: %s", result, input)
	}
}

func TestProcess_Flowchart(t *testing.T) {
	input := `# Test

` + "```mermaid" + `
graph LR
    A --> B
` + "```" + `

More text.`

	result := Process(input, true)

	// Should not contain the original mermaid code
	if strings.Contains(result, "graph LR") {
		t.Error("mermaid code should be replaced with ASCII")
	}

	// Should contain box-drawing characters from the rendered diagram
	if !strings.Contains(result, "─") && !strings.Contains(result, "-") {
		t.Error("should contain horizontal line characters")
	}

	// Should preserve surrounding markdown
	if !strings.Contains(result, "# Test") {
		t.Error("should preserve markdown before mermaid block")
	}
	if !strings.Contains(result, "More text.") {
		t.Error("should preserve markdown after mermaid block")
	}
}

func TestProcess_SequenceDiagram(t *testing.T) {
	input := "```mermaid\nsequenceDiagram\n    Alice->>Bob: Hello\n```"

	result := Process(input, true)

	// Should not contain the original sequenceDiagram keyword
	if strings.Contains(result, "sequenceDiagram") {
		t.Error("sequence diagram code should be replaced with ASCII")
	}

	// Should contain participant names in the rendered output
	if !strings.Contains(result, "Alice") || !strings.Contains(result, "Bob") {
		t.Error("should contain participant names in rendered output")
	}
}

func TestProcess_EmptyBlock(t *testing.T) {
	input := "```mermaid\n```"
	result := Process(input, true)

	// Empty blocks should be preserved unchanged
	if result != input {
		t.Errorf("empty mermaid block should be preserved\ngot: %s\nwant: %s", result, input)
	}
}

func TestProcess_InvalidSyntax(t *testing.T) {
	input := "```mermaid\ninvalid syntax here\n```"
	result := Process(input, true)

	// Should preserve original block unchanged
	if result != input {
		t.Errorf("invalid syntax should preserve original block unchanged\ngot: %s\nwant: %s", result, input)
	}
}

func TestProcess_Subgraph(t *testing.T) {
	input := `# Test

` + "```mermaid" + `
graph LR
    subgraph Group1
        A[Node A]
    end
    A --> B
` + "```" + `

More text.`

	result := Process(input, true)

	// Subgraphs should be rendered (mermaid-ascii supports basic subgraphs)
	if strings.Contains(result, "subgraph") {
		t.Error("subgraph should be rendered, not shown as raw code")
	}

	// Should preserve surrounding markdown
	if !strings.Contains(result, "# Test") {
		t.Error("should preserve markdown before mermaid block")
	}
}

func TestProcess_UnsupportedDiagramType(t *testing.T) {
	input := "```mermaid\npie\n    \"A\" : 50\n    \"B\" : 50\n```"
	result := Process(input, true)

	// Should preserve original block unchanged
	if result != input {
		t.Errorf("unsupported diagram type should preserve original\ngot: %s\nwant: %s", result, input)
	}
}

func TestProcess_MultipleBlocks(t *testing.T) {
	input := `# Doc

` + "```mermaid" + `
graph LR
    A --> B
` + "```" + `

Some text.

` + "```mermaid" + `
graph TD
    C --> D
` + "```"

	result := Process(input, true)

	// Should not contain any of the original mermaid code (both are simple, supported graphs)
	if strings.Contains(result, "graph LR") || strings.Contains(result, "graph TD") {
		t.Error("all mermaid blocks should be replaced")
	}

	// Should contain text between blocks
	if !strings.Contains(result, "Some text.") {
		t.Error("text between blocks should be preserved")
	}
}

func TestProcess_MixedSupportedAndUnsupported(t *testing.T) {
	input := `# Doc

` + "```mermaid" + `
graph LR
    A --> B
` + "```" + `

` + "```mermaid" + `
sequenceDiagram
    loop Every minute
        A->>B: Check
    end
` + "```"

	result := Process(input, true)

	// Simple graph should be rendered (no "graph LR")
	if strings.Contains(result, "graph LR") {
		t.Error("simple graph should be rendered")
	}

	// Loop (unsupported sequence feature) should be preserved
	if !strings.Contains(result, "loop Every minute") {
		t.Error("unsupported loop should be preserved unchanged")
	}
}

func TestProcess_MixedCodeBlocks(t *testing.T) {
	input := `# Doc

` + "```go" + `
func main() {}
` + "```" + `

` + "```mermaid" + `
graph LR
    A --> B
` + "```" + `

` + "```python" + `
print("hello")
` + "```"

	result := Process(input, true)

	// Should preserve non-mermaid code blocks
	if !strings.Contains(result, "```go") {
		t.Error("should preserve go code block marker")
	}
	if !strings.Contains(result, "func main()") {
		t.Error("should preserve go code content")
	}
	if !strings.Contains(result, "```python") {
		t.Error("should preserve python code block marker")
	}
	if !strings.Contains(result, "print(") {
		t.Error("should preserve python code content")
	}

	// Should replace mermaid block
	if strings.Contains(result, "graph LR") {
		t.Error("mermaid block should be replaced")
	}
}
