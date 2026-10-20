package utils

import (
	"strings"
	"sync"
	"testing"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
)

func TestGlamourStyleChromaFormatter(t *testing.T) {
	render := func(t *testing.T) string {
		t.Helper()
		r, err := glamour.NewTermRenderer(GlamourStyle(styles.DarkStyle, false))
		if err != nil {
			t.Fatal(err)
		}
		out, err := r.Render("```go\nfunc main() {}\n```\n")
		if err != nil {
			t.Fatal(err)
		}
		return out
	}

	t.Setenv("TTY_FORCE", "1")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TMUX", "")

	chromaFormatter = sync.OnceValue(detectChromaFormatter)
	t.Setenv("COLORTERM", "")
	if out := render(t); strings.Contains(out, "\x1b[38;2;") {
		t.Errorf("expected 256-color code block, got %q", out)
	}

	chromaFormatter = sync.OnceValue(detectChromaFormatter)
	t.Setenv("COLORTERM", "truecolor")
	if out := render(t); !strings.Contains(out, "\x1b[38;2;") {
		t.Errorf("expected true color code block, got %q", out)
	}
}
