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

func TestGlamourRenderAlerts(t *testing.T) {
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

	md := "> [!NOTE]\n> Useful information.\n\n> [!WARNING]\n> Careful.\n"
	out, err := glamourRender(m, md)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out, "Note") || !strings.Contains(out, "Warning") {
		t.Errorf("expected alert titles in pager output, got %q", out)
	}
	if strings.Contains(out, "[!NOTE]") || strings.Contains(out, "[!WARNING]") {
		t.Errorf("expected alert markers to be consumed, got %q", out)
	}
}

func TestGlamourRenderAlertsNoWrap(t *testing.T) {
	common := &commonModel{
		cfg: Config{
			GlamourEnabled: true,
			GlamourStyle:   "dark",
		},
		styles: newStyles(true),
		width:  0,
		height: 24,
	}
	m := newPagerModel(common)
	m.currentDocument = markdown{Note: "test.md"}
	m.setSize(0, 24)

	out, err := glamourRender(m, "> [!NOTE]\n> Useful information.\n")
	if err != nil {
		t.Fatal(err)
	}

	// Words are styled separately at zero width, so don't look for a whole
	// sentence here.
	if !strings.Contains(out, "Note") || !strings.Contains(out, "Useful") {
		t.Errorf("expected the alert in pager output, got %q", out)
	}
}
