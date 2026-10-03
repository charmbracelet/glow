package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestExecuteCLIRendersMMDAsMarkdown(t *testing.T) {
	const content = "# Title\n\nHello **world**\n"

	oldStyle, oldWidth := style, width
	oldPager, oldTUI := pager, tui
	t.Cleanup(func() {
		style, width = oldStyle, oldWidth
		pager, tui = oldPager, oldTUI
	})

	style = "notty"
	width = 80
	pager = false
	tui = false

	render := func(name string) string {
		t.Helper()

		var out bytes.Buffer
		src := &source{
			reader: io.NopCloser(strings.NewReader(content)),
			URL:    name,
		}
		if err := executeCLI(&cobra.Command{}, src, &out); err != nil {
			t.Fatalf("executeCLI(%q): %v", name, err)
		}
		return out.String()
	}

	if got, want := render("test.mmd"), render("test.md"); got != want {
		t.Fatalf(".mmd rendered differently than .md\n.mmd:\n%q\n.md:\n%q", got, want)
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
