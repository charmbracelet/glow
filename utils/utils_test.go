package utils

import (
	"testing"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
)

// renderCode renders a fenced code block with the given style in code mode.
func renderCode(t *testing.T, style string) string {
	t.Helper()
	r, err := glamour.NewTermRenderer(GlamourStyle(style, true))
	if err != nil {
		t.Fatalf("new renderer for %q: %v", style, err)
	}
	out, err := r.Render("```go\nfmt.Println(\"hello\")\n```\n")
	if err != nil {
		t.Fatalf("render with %q: %v", style, err)
	}
	return out
}

// TestGlamourStyleTokyoNightCodeDistinctFromDracula guards against the
// tokyo-night code style being mapped to the dracula StyleConfig. The two
// themes use different colors, so their rendered code output must differ.
func TestGlamourStyleTokyoNightCodeDistinctFromDracula(t *testing.T) {
	t.Setenv("CLICOLOR_FORCE", "1") // force ANSI color so styles are distinguishable

	tokyo := renderCode(t, styles.TokyoNightStyle)
	dracula := renderCode(t, styles.DraculaStyle)

	if tokyo == dracula {
		t.Errorf("tokyo-night code rendering is identical to dracula; GlamourStyle must use TokyoNightStyleConfig")
	}
}
