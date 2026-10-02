package ui

import (
	"strings"
	"testing"
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

func TestGlamourRenderMermaid(t *testing.T) {
	t.Setenv("LC_ALL", "en_US.UTF-8")

	md := "# Plan\n\n```mermaid\nflowchart LR\nA[Build] --> B{Gate}\nB -->|yes| C[Ship]\n```\n"
	newTestPager := func(mermaid bool) pagerModel {
		common := &commonModel{
			cfg: Config{
				GlamourEnabled:  true,
				GlamourStyle:    "dark",
				GlamourMaxWidth: 80,
				Mermaid:         mermaid,
			},
			styles: newStyles(true),
			width:  80,
			height: 24,
		}
		m := newPagerModel(common)
		m.currentDocument = markdown{Note: "test.md"}
		m.setSize(80, 24)
		return m
	}

	out, err := glamourRender(newTestPager(true), md)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "┌") || !strings.Contains(out, "Build") {
		t.Errorf("expected a rendered diagram, got %q", out)
	}
	if strings.Contains(out, "flowchart LR") {
		t.Errorf("expected the source not to be shown, got %q", out)
	}

	out, err = glamourRender(newTestPager(false), md)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "flowchart LR") {
		t.Errorf("expected the source to be shown when disabled, got %q", out)
	}
	if strings.ContainsAny(out, "┌┐└┘") {
		t.Errorf("expected no diagram when disabled, got %q", out)
	}
}
