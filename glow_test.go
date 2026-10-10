package main

import (
	"bytes"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

var osc8Sequence = regexp.MustCompile(`\x1b]8;[^\x07]*(\x07|\x1b\\)`)

func stripOSC8Sequences(s string) string {
	return osc8Sequence.ReplaceAllString(s, "")
}

func TestExecuteCLIPrintsTableImageLinksWithoutWidthLimit(t *testing.T) {
	t.Setenv("GLOW_TEXT_SIZING", "off")

	oldWidth, oldStyle, oldPager, oldTUI := width, style, pager, tui
	t.Cleanup(func() {
		width, style, pager, tui = oldWidth, oldStyle, oldPager, oldTUI
	})
	width, style, pager, tui = 0, "notty", false, false

	const imageURL = "https://example.com/image.png?size=14"
	markdown := "| Name |\n| --- |\n| ![image](" + imageURL + ") |\n"
	var output bytes.Buffer
	src := &source{
		reader: io.NopCloser(strings.NewReader(markdown)),
		URL:    "https://example.com/README.md",
	}
	if err := executeCLI(&cobra.Command{}, src, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stripOSC8Sequences(output.String()), imageURL) {
		t.Fatalf("expected the full image URL to be visible in output, got %q", output.String())
	}
}

func TestGlowFlags(t *testing.T) {
	tt := []struct {
		args  []string
		check func() bool
	}{
		{
			args: []string{"-p"},
			check: func() bool {
				return pager
			},
		},
		{
			args: []string{"-s", "light"},
			check: func() bool {
				return style == "light"
			},
		},
		{
			args: []string{"-w", "40"},
			check: func() bool {
				return width == 40
			},
		},
	}

	for _, v := range tt {
		err := rootCmd.ParseFlags(v.args)
		if err != nil {
			t.Fatal(err)
		}
		if !v.check() {
			t.Errorf("Parsing flag failed: %s", v.args)
		}
	}
}
