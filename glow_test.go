package main

import (
	"bytes"
	"io"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/viper"
)

var cliTestMu sync.Mutex

func TestGlowFlags(t *testing.T) {
	cliTestMu.Lock()
	defer cliTestMu.Unlock()

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

func TestCLIPreserveNewLinesFlag(t *testing.T) {
	cliTestMu.Lock()
	defer cliTestMu.Unlock()

	md := "soft\nbreak\n"

	render := func(preserve bool) string {
		oldPreserve := preserveNewLines
		oldStyle := style
		oldWidth := width
		oldPager := pager
		oldTUI := tui
		t.Cleanup(func() {
			preserveNewLines = oldPreserve
			style = oldStyle
			width = oldWidth
			pager = oldPager
			tui = oldTUI
		})

		preserveNewLines = preserve
		style = "notty"
		width = 80
		pager = false
		tui = false
		viper.Set("pager", false)
		viper.Set("tui", false)
		_ = rootCmd.Flags().Set("pager", "false")
		_ = rootCmd.Flags().Set("tui", "false")
		rootCmd.Flags().Lookup("pager").Changed = false
		rootCmd.Flags().Lookup("tui").Changed = false

		var buf bytes.Buffer
		src := &source{reader: io.NopCloser(strings.NewReader(md)), URL: "test.md"}
		if err := executeCLI(rootCmd, src, &buf); err != nil {
			t.Fatalf("executeCLI failed: %v", err)
		}
		return ansiStrip.ReplaceAllString(buf.String(), "")
	}

	outFalse := render(false)
	outTrue := render(true)
	if strings.Contains(outFalse, "soft\nbreak") {
		t.Fatalf("expected soft line break to collapse when preserve-new-lines is false, got: %q", outFalse)
	}
	if outFalse == outTrue {
		t.Fatalf("expected preserve-new-lines flag to change rendering")
	}
}

var ansiStrip = regexp.MustCompile(`\x1b\[[0-9;]*m`)
