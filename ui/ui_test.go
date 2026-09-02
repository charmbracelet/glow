package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/muesli/gitcha"
)

func TestLocalFileToMarkdownSkipsBrokenSymlink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.md")

	err := os.Symlink(filepath.Join(dir, "missing.md"), path)
	if err != nil {
		t.Skipf("symlinks are not available: %v", err)
	}

	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("expected to stat symlink: %v", err)
	}

	md := localFileToMarkdown(dir, gitcha.SearchResult{
		Path: path,
		Info: info,
	})

	if md != nil {
		t.Fatalf("expected broken symlink to be skipped, got note %q", md.Note)
	}
}
