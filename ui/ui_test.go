package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestLocalFileSearchRootUsesParentDirectoryForFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "index.md")
	if err := os.WriteFile(path, []byte("# Test\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := localFileSearchRoot(path)
	if err != nil {
		t.Fatal(err)
	}

	if got != dir {
		t.Fatalf("expected %q, got %q", dir, got)
	}
}

func TestUnloadDocumentStartsSearchWhenListingWasNotLoaded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "index.md")
	if err := os.WriteFile(path, []byte("# Test\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	initSections()
	common := &commonModel{
		cfg: Config{
			Path:         path,
			ShowAllFiles: true,
		},
	}
	m := model{
		common: common,
		state:  stateShowDocument,
		stash:  newStashModel(common),
		pager:  newPagerModel(common),
	}
	t.Cleanup(func() {
		if err := m.pager.watcher.Close(); err != nil {
			t.Fatal(err)
		}
	})

	cmds := m.unloadDocument()
	if len(cmds) == 0 {
		t.Fatal("expected unload to start a local file search")
	}

	var msg tea.Msg
	for _, cmd := range cmds {
		msg = cmd()
		if _, ok := msg.(initLocalFileSearchMsg); ok {
			break
		}
	}

	got, ok := msg.(initLocalFileSearchMsg)
	if !ok {
		t.Fatalf("expected initLocalFileSearchMsg, got %T", msg)
	}
	if got.cwd != dir {
		t.Fatalf("expected search cwd %q, got %q", dir, got.cwd)
	}
	for range got.ch {
	}
}
