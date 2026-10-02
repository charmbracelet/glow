package utils

import (
	"strings"
	"testing"
)

func TestReplaceMermaidBlocks(t *testing.T) {
	md := "# Title\n\n```mermaid\nflowchart TD\n    A --> B\n```\n\nSome text.\n"
	out := ReplaceMermaidBlocks(md)
	if !strings.Contains(out, "│A│") || !strings.Contains(out, "▼") {
		t.Errorf("mermaid block not rendered:\n%s", out)
	}
	if strings.Contains(out, "flowchart") {
		t.Errorf("raw source leaked:\n%s", out)
	}
	if !strings.HasPrefix(out, "# Title\n\n") {
		t.Errorf("surrounding content changed:\n%q", out)
	}
}

func TestReplaceKeepsUnparseableBlocks(t *testing.T) {
	md := "```mermaid\nsequenceDiagram\n    A->>B: hi\n```\n"
	if out := ReplaceMermaidBlocks(md); out != md {
		t.Errorf("unparseable block was changed:\n%s", out)
	}
}

func TestReplaceIgnoresOtherFences(t *testing.T) {
	md := "```go\nfmt.Println(1)\n```\n\n```mermaid\nflowchart TD\n    A --> B\n```\n"
	out := ReplaceMermaidBlocks(md)
	if !strings.Contains(out, "fmt.Println(1)") {
		t.Errorf("go block changed:\n%s", out)
	}
	if !strings.Contains(out, "│A│") {
		t.Errorf("mermaid block not rendered:\n%s", out)
	}
}

func TestReplaceMermaidInsideFence(t *testing.T) {
	// a mermaid example inside a markdown fence must stay verbatim
	md := "~~~markdown\n```mermaid\nflowchart TD\n    A --> B\n```\n~~~\n"
	if out := ReplaceMermaidBlocks(md); out != md {
		t.Errorf("nested block was changed:\n%s", out)
	}
}

func TestReplaceUnclosedFence(t *testing.T) {
	md := "text\n\n```mermaid\nflowchart TD\n    A --> B\n"
	if out := ReplaceMermaidBlocks(md); out != md {
		t.Errorf("unclosed fence changed:\n%s", out)
	}
}
