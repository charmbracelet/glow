package utils

import (
	"strings"
	"sync"
	"testing"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
)

func renderPlainAndSized(t *testing.T, markdown string, wordWrap int) (string, string) {
	t.Helper()

	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(styles.DarkStyleConfig),
		glamour.WithWordWrap(wordWrap),
	)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := r.Render(markdown)
	if err != nil {
		t.Fatal(err)
	}

	cfg := styles.DarkStyleConfig
	AddHeadingSizeMarkers(&cfg)
	r, err = glamour.NewTermRenderer(
		glamour.WithStyles(cfg),
		glamour.WithWordWrap(wordWrap),
	)
	if err != nil {
		t.Fatal(err)
	}
	marked, err := r.Render(markdown)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(marked, markerPrefix) {
		t.Fatal("rendered output does not contain heading markers")
	}

	return plain, ApplyTextSizing(marked)
}

func TestApplyTextSizingPassthrough(t *testing.T) {
	in := "hello \x1b[1mworld\x1b[0m"
	if got := ApplyTextSizing(in); got != in {
		t.Errorf("expected output unchanged, got %q", got)
	}
}

func TestApplyTextSizingLevels(t *testing.T) {
	_, sized := renderPlainAndSized(t, "# Hello\n\n## World\n\n### Third\n", 80)

	if !strings.Contains(sized, "\x1b]66;s=3;Hello\x1b\\") {
		t.Errorf("expected H1 text at 3x scale, got %q", sized)
	}
	if !strings.Contains(sized, "\x1b]66;s=3; \x1b\\") {
		t.Errorf("expected H1 prefix space at 3x scale, got %q", sized)
	}
	if !strings.Contains(sized, "\x1b]66;s=2;World\x1b\\") {
		t.Errorf("expected H2 text at 2x scale, got %q", sized)
	}
	if !strings.Contains(sized, "\x1b]66;s=2:n=3:d=4:w=3;Thir\x1b\\") {
		t.Errorf("expected H3 chunks at tight 1.5x scale, got %q", sized)
	}
	if !strings.Contains(sized, "\x1b]66;s=2:n=3:d=4:w=1;d\x1b\\") {
		t.Errorf("expected H3 remainder chunk at tight 1.5x scale, got %q", sized)
	}
	if strings.Contains(sized, markerPrefix) {
		t.Errorf("expected all markers to be replaced, got %q", sized)
	}
}

func TestApplyTextSizingCompensationLines(t *testing.T) {
	plain, sized := renderPlainAndSized(t, "# Hello\n\n## World\n\n### Third\n\nparagraph\n", 80)

	if diff := strings.Count(sized, "\n") - strings.Count(plain, "\n"); diff != 4 {
		t.Errorf("expected 4 extra newlines (2 for H1, 1 each for H2/H3), got %d", diff)
	}
}

func TestApplyTextSizingWrappedHeading(t *testing.T) {
	plain, sized := renderPlainAndSized(t, "# aaa bbb ccc\n\npara\n", 8)

	plainLines := strings.Split(plain, "\n")
	sizedLines := strings.Split(sized, "\n")

	plainHead := lineIndex(t, plainLines, "aaa")
	plainPara := lineIndex(t, plainLines, "para")
	sizedHead := lineIndex(t, sizedLines, "aaa")
	sizedPara := lineIndex(t, sizedLines, "para")

	headLines := plainPara - plainHead - 1
	if headLines < 2 {
		t.Fatalf("expected heading to wrap, got %d line(s)", headLines)
	}
	if got, want := sizedPara-sizedHead, headLines*3+1; got != want {
		t.Errorf("expected wrapped heading to occupy %d rows, got %d", want, got)
	}
}

func TestApplyTextSizingNestedStyles(t *testing.T) {
	_, sized := renderPlainAndSized(t, "# Hello *World*\n", 80)

	if !strings.Contains(sized, "\x1b]66;s=3;Hello \x1b\\") {
		t.Errorf("expected first chunk at 3x scale, got %q", sized)
	}
	if !strings.Contains(sized, "\x1b]66;s=3;World\x1b\\") {
		t.Errorf("expected emphasized chunk at 3x scale, got %q", sized)
	}
}

func TestDetectTextSizingEnvOverride(t *testing.T) {
	t.Setenv("GLOW_TEXT_SIZING", "off")
	if detectTextSizing() {
		t.Error("expected detection to be disabled")
	}
	t.Setenv("GLOW_TEXT_SIZING", "on")
	if !detectTextSizing() {
		t.Error("expected detection to be forced on")
	}
}

func TestGlamourStyleTextSizing(t *testing.T) {
	render := func(t *testing.T) string {
		t.Helper()
		r, err := glamour.NewTermRenderer(
			GlamourStyle(styles.DarkStyle, false),
			glamour.WithWordWrap(80),
		)
		if err != nil {
			t.Fatal(err)
		}
		out, err := r.Render("# Hello\n")
		if err != nil {
			t.Fatal(err)
		}
		return out
	}

	textSizingEnabled = sync.OnceValue(detectTextSizing)
	t.Setenv("GLOW_TEXT_SIZING", "off")
	if out := render(t); strings.Contains(out, markerPrefix) {
		t.Errorf("expected no heading markers when disabled, got %q", out)
	}

	textSizingEnabled = sync.OnceValue(detectTextSizing)
	t.Setenv("GLOW_TEXT_SIZING", "on")
	out := render(t)
	if !strings.Contains(out, markerOpen(1)) {
		t.Errorf("expected heading markers when enabled, got %q", out)
	}
	if sized := ApplyTextSizing(out); !strings.Contains(sized, "\x1b]66;s=3;") {
		t.Errorf("expected OSC 66 sequences after ApplyTextSizing, got %q", sized)
	}
}

func TestScanEscape(t *testing.T) {
	tt := []struct{ in, want string }{
		{"\x1b[1;31mfoo", "\x1b[1;31m"},
		{"\x1b[0m", "\x1b[0m"},
		{"\x1b]66;s=2;hi\a rest", "\x1b]66;s=2;hi\a"},
		{"\x1b]6666;1\x1b\\x", "\x1b]6666;1\x1b\\"},
		{"\x1b(B", "\x1b(B"},
	}
	for _, tc := range tt {
		if got := scanEscape(tc.in); got != tc.want {
			t.Errorf("scanEscape(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func lineIndex(t *testing.T, lines []string, substr string) int {
	t.Helper()
	for i, ln := range lines {
		if strings.Contains(ln, substr) {
			return i
		}
	}
	t.Fatalf("no line contains %q", substr)
	return -1
}
