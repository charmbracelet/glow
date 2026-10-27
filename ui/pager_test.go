package ui

import (
	"strings"
	"testing"
)

func TestPagerWrapsLinesWiderThanViewport(t *testing.T) {
	common := &commonModel{
		cfg:    Config{},
		styles: newStyles(true),
		width:  10,
		height: 4,
	}
	m := newPagerModel(common)
	m.setSize(10, 4)
	m.setContent("one\ntwo\n0123456789abcdefghij\nthree\nfour")

	// The over-wide line occupies two rows, so the viewport has six rows in
	// total and scrolling must not skip any of them.
	if got, want := m.viewport.TotalLineCount(), 6; got != want {
		t.Fatalf("expected %d lines, got %d", want, got)
	}

	m.viewport.GotoBottom()
	if got := m.viewport.View(); !strings.Contains(got, "cdefghij") {
		t.Errorf("expected the wrapped end of the long line to be visible, got %q", got)
	}
}

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
