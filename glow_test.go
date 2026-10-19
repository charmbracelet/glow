package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

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

func TestRootCmdAcceptsMultipleArgs(t *testing.T) {
	if err := rootCmd.Args(rootCmd, []string{"a.md", "b.md", "c.md"}); err != nil {
		t.Errorf("expected multiple args to be accepted, got %v", err)
	}
}

func TestExecuteArgRendersMultipleFiles(t *testing.T) {
	// Reset globals mutated by TestGlowFlags, and use a fresh command so
	// rootCmd's flag state (e.g. Changed("pager")) doesn't leak into this test.
	pager = false
	tui = false
	style = "auto"
	width = 80
	cmd := &cobra.Command{}

	dir := t.TempDir()
	var paths []string
	for i, name := range []string{"a.md", "b.md"} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(fmt.Sprintf("# Doc %d\n", i+1)), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}

	var buf bytes.Buffer
	for _, arg := range paths {
		if err := executeArg(cmd, arg, &buf); err != nil {
			t.Fatal(err)
		}
	}

	out := buf.String()
	// Strip ANSI escape sequences before matching, since glamour styles
	// each word individually.
	out = ansiPattern.ReplaceAllString(out, "")
	if !strings.Contains(out, "Doc 1") || !strings.Contains(out, "Doc 2") {
		t.Errorf("expected both documents to be rendered, got %q", out)
	}
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)
