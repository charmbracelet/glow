package ui

import (
	"regexp"
	"strings"
	"testing"
)

var osc8Sequence = regexp.MustCompile(`\x1b]8;[^\x07]*(\x07|\x1b\\)`)

func stripOSC8Sequences(s string) string {
	return osc8Sequence.ReplaceAllString(s, "")
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

func TestGlamourRenderPrintsTableLinksWithoutWidthLimit(t *testing.T) {
	t.Setenv("GLOW_TEXT_SIZING", "off")

	const imageURL = "https://example.com/image.png?size=14"
	md := "| Name |\n| --- |\n| ![image](" + imageURL + ") |\n"
	common := &commonModel{
		cfg: Config{
			GlamourEnabled:  true,
			GlamourStyle:    "notty",
			GlamourMaxWidth: 0,
		},
		styles: newStyles(true),
		width:  80,
		height: 24,
	}
	m := newPagerModel(common)
	m.currentDocument = markdown{Note: "test.md"}
	m.setSize(80, 24)

	out, err := glamourRender(m, md)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stripOSC8Sequences(out), imageURL) {
		t.Fatalf("expected the full image URL to be visible in output, got %q", out)
	}
}
