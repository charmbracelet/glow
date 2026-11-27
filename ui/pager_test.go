package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestGlamourRenderTextSizing(t *testing.T) {
	t.Setenv("GLOW_TEXT_SIZING", "on")

	common := &commonModel{
		cfg: Config{
			GlamourEnabled:  true,
			GlamourStyle:    "dark",
			GlamourMaxWidth: 80,
		},
		styles: newStyles(true),
		width:  80,
		height: 24,
	}
	m := newPagerModel(common)
	m.currentDocument = markdown{Note: "test.md"}
	m.setSize(80, 24)

	out, err := glamourRender(m, "# Hello\n\n## World\n")
	if err != nil {
		t.Fatal(err)
	}
	// The dark style's H1 has a background color, which renders broken when
	// scaled, so it is left at normal size.
	if strings.Contains(out, "\x1b]66;s=3;") {
		t.Errorf("expected H1 with background to not be scaled, got %q", out)
	}
	if !strings.Contains(out, "Hello") {
		t.Errorf("expected H1 text in pager output, got %q", out)
	}
	if !strings.Contains(out, "\x1b]66;s=2;World\x1b\\") {
		t.Errorf("expected H2 at 2x scale in pager output, got %q", out)
	}
	if strings.Contains(out, "\x1b]6666;") {
		t.Errorf("expected no leftover markers in pager output, got %q", out)
	}
}

func TestGlamourRenderInlineLinks(t *testing.T) {
	common := &commonModel{
		cfg: Config{
			GlamourEnabled:  true,
			GlamourStyle:    "dark",
			GlamourMaxWidth: 80,
			InlineLinks:     true,
		},
		styles: newStyles(true),
		width:  80,
		height: 24,
	}
	m := newPagerModel(common)
	m.currentDocument = markdown{Note: "test.md"}
	m.setSize(80, 24)

	const md = "[Prose](https://example.com/prose), then <https://example.com/auto>." +
		"\n\n| Link |\n| --- |\n| [Table](https://example.com/table) |\n"
	out, err := glamourRender(m, md)
	if err != nil {
		t.Fatal(err)
	}
	visible := ansi.Strip(out)
	if !strings.Contains(visible, "Prose, then https://example.com/auto.") ||
		!strings.Contains(visible, "Table") {
		t.Fatalf("expected link labels and the bare autolink, got %q", visible)
	}
	for _, target := range []string{"https://example.com/prose", "https://example.com/table"} {
		if strings.Contains(visible, target) || !strings.Contains(out, ";"+target+"\x07") {
			t.Errorf("expected %s only in an OSC 8 destination, got %q", target, out)
		}
	}
	if strings.Contains(visible, "[1]") {
		t.Errorf("unexpected table link reference in %q", visible)
	}
}
