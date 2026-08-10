package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePagerScript(t *testing.T, dir, name, body string) string {
	t.Helper()
	script := filepath.Join(dir, name)
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return script
}

func TestRunPagerCommandUsesTempFile(t *testing.T) {
	dir := t.TempDir()
	script := writePagerScript(t, dir, "pager.sh", `#!/bin/sh
if [ ! -f "$1" ]; then
  echo "missing pager file arg" >&2
  exit 2
fi
cat "$1"
`)

	t.Setenv("PAGER", script)
	if err := runPagerCommand("rendered markdown\n"); err != nil {
		t.Fatalf("runPagerCommand: %v", err)
	}
}

func TestRunPagerCommandEmptyContent(t *testing.T) {
	dir := t.TempDir()
	script := writePagerScript(t, dir, "pager.sh", `#!/bin/sh
test -f "$1"
`)

	t.Setenv("PAGER", script)
	if err := runPagerCommand(""); err != nil {
		t.Fatalf("runPagerCommand: %v", err)
	}
}

func TestRunPagerCommandRespectsPagerFlags(t *testing.T) {
	dir := t.TempDir()
	script := writePagerScript(t, dir, "pager.sh", `#!/bin/sh
if [ "$1" != "-r" ] || [ ! -f "$2" ]; then
  echo "unexpected pager args: $*" >&2
  exit 2
fi
cat "$2"
`)

	t.Setenv("PAGER", script+" -r")
	if err := runPagerCommand("flagged pager\n"); err != nil {
		t.Fatalf("runPagerCommand: %v", err)
	}
}

func TestRunPagerCommandIncludesPagerStderr(t *testing.T) {
	dir := t.TempDir()
	script := writePagerScript(t, dir, "pager.sh", `#!/bin/sh
echo "pager rejected input" >&2
exit 1
`)

	t.Setenv("PAGER", script)
	err := runPagerCommand("content\n")
	if err == nil {
		t.Fatal("expected pager failure")
	}
	if !strings.Contains(err.Error(), "pager rejected input") {
		t.Fatalf("expected pager stderr in error, got: %v", err)
	}
}
