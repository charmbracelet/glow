package utils

import (
	"strings"
	"testing"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
)

func TestGlamourStyleTableRowBorders(t *testing.T) {
	markdown := strings.TrimSpace(`
| Header A | Header B |
| -------- | -------- |
| Cell 1   | Cell 2   |
| Cell 3   | Cell 4   |
| Cell 5   | Cell 6   |
`)

	render := func(tableRowBorders bool) string {
		t.Helper()
		r, err := glamour.NewTermRenderer(
			GlamourStyle(styles.AsciiStyle, false, tableRowBorders),
			glamour.WithWordWrap(80),
		)
		if err != nil {
			t.Fatal(err)
		}
		out, err := r.Render(markdown)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}

	dashLines := func(rendered string) int {
		n := 0
		for _, line := range strings.Split(rendered, "\n") {
			if strings.Contains(line, "---") {
				n++
			}
		}
		return n
	}

	if got, want := dashLines(render(false)), 1; got != want {
		t.Fatalf("default rules = %d, want %d", got, want)
	}
	if got, want := dashLines(render(true)), 3; got != want {
		t.Fatalf("table row borders rules = %d, want %d", got, want)
	}
}
