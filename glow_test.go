package main

import (
	"os"
	"testing"
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

// TestTerminalWidthFromFdRejectsPipe ensures that a pipe (the canonical
// non-tty fd type) is reported as "not a terminal" rather than e.g.
// returning a stale or zero width as success. This is the regression we
// care about for `glow x.md | less`: the pager's pipe must not be
// mistaken for a terminal.
func TestTerminalWidthFromFdRejectsPipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	t.Cleanup(func() {
		_ = r.Close()
		_ = w.Close()
	})

	for name, fd := range map[string]uintptr{"read end": r.Fd(), "write end": w.Fd()} {
		if gotW, ok := terminalWidthFromFd(fd); ok {
			t.Errorf("%s: expected (_, false), got (%d, true)", name, gotW)
		}
	}
}

// TestTerminalWidthFromFdRejectsRegularFile guards the other common
// non-tty case: stdout redirected to a file (`glow x.md > out`).
func TestTerminalWidthFromFdRejectsRegularFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "glow-width-*")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })

	if gotW, ok := terminalWidthFromFd(f.Fd()); ok {
		t.Errorf("regular file: expected (_, false), got (%d, true)", gotW)
	}
}
