package main

import (
	"os"
	"path/filepath"
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

// writePagerScript writes an executable shell script and returns its path.
func writePagerScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pager")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o600); err != nil { //nolint:gosec
		t.Fatalf("unable to write pager script: %v", err)
	}
	if err := os.Chmod(path, 0o700); err != nil { //nolint:gosec
		t.Fatalf("unable to make pager script executable: %v", err)
	}
	return path
}

// TestRunPagerFeedsStdin checks the pager receives the rendered output.
func TestRunPagerFeedsStdin(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "pager-input")
	out := "rendered output\n"

	script := writePagerScript(t, "cat > "+dest)
	if err := runPager(script, out); err != nil {
		t.Fatalf("unable to run pager: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("unable to read pager input: %v", err)
	}
	if string(got) != out {
		t.Errorf("expected pager input %q, got %q", out, got)
	}
}

// TestRunPagerPropagatesStderr checks the pager inherits glow's stderr.
//
// os/exec sends a child's stderr to the null device when Cmd.Stderr is unset,
// which leaves pagers like less unable to write their prompt or query the
// terminal, corrupting their rendering when scrolling backwards.
func TestRunPagerPropagatesStderr(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("unable to create pipe: %v", err)
	}
	defer r.Close() //nolint:errcheck

	origStderr := os.Stderr
	os.Stderr = w
	runErr := runPager(writePagerScript(t, `echo pager-was-here >&2`), "")
	os.Stderr = origStderr
	_ = w.Close()

	if runErr != nil {
		t.Fatalf("unable to run pager: %v", runErr)
	}

	buf := make([]byte, 64)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("unable to read glow stderr: %v", err)
	}
	if got := string(buf[:n]); got != "pager-was-here\n" {
		t.Errorf("expected pager stderr %q, got %q", "pager-was-here\n", got)
	}
}

// TestRunPagerRejectsUnparsableCommand checks the error path is preserved.
func TestRunPagerRejectsUnparsableCommand(t *testing.T) {
	if err := runPager("'", ""); err == nil {
		t.Fatal("expected an error for an unparsable pager command")
	}
}
