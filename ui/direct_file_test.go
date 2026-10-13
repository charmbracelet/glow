package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func testDocument(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "index.md")
	if err := os.WriteFile(path, []byte("# Test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir, path
}

func checkSearch(t *testing.T, msg tea.Msg, dir, path string) {
	t.Helper()
	search, ok := msg.(initLocalFileSearchMsg)
	if !ok {
		t.Fatalf("expected initLocalFileSearchMsg, got %T: %v", msg, msg)
	}
	if search.ch == nil {
		t.Fatal("search returned a nil channel")
	}

	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	var paths []string
	for {
		select {
		case result, open := <-search.ch:
			if !open {
				if search.cwd != dir {
					t.Fatalf("expected search root %q, got %q", dir, search.cwd)
				}
				if len(paths) != 1 || paths[0] != path {
					t.Fatalf("expected only %q, got %v", path, paths)
				}
				return
			}
			paths = append(paths, result.Path)
		case <-timer.C:
			t.Fatal("search did not finish within two seconds")
		}
	}
}

func TestFindLocalFilesFromFile(t *testing.T) {
	dir, path := testDocument(t)
	common := commonModel{cfg: Config{Path: path, ShowAllFiles: true}}
	// Repeating the same search also covers the file-path input used by reload.
	for range 2 {
		checkSearch(t, findLocalFiles(common)(), dir, path)
	}
}

func TestFindLocalFilesFromDirectory(t *testing.T) {
	dir, path := testDocument(t)
	common := commonModel{cfg: Config{Path: dir, ShowAllFiles: true}}
	checkSearch(t, findLocalFiles(common)(), dir, path)
}

func TestFindLocalFilesMissingPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.md")
	msg := findLocalFiles(commonModel{cfg: Config{Path: path}})()
	if _, ok := msg.(errMsg); !ok {
		t.Fatalf("expected errMsg, got %T", msg)
	}
}

func TestUnloadDirectDocumentStartsSearch(t *testing.T) {
	dir, path := testDocument(t)
	m := newModel(Config{Path: path, ShowAllFiles: true}, "").(model)
	if m.pager.watcher == nil {
		t.Fatal("pager watcher was not initialized")
	}
	t.Cleanup(func() {
		if err := m.pager.watcher.Close(); err != nil {
			t.Error(err)
		}
	})
	if m.state != stateShowDocument {
		t.Fatal("expected direct document mode")
	}
	cmds := m.unloadDocument()
	if m.state != stateShowStash {
		t.Fatal("expected listing after unloading")
	}
	if len(cmds) == 0 {
		t.Fatal("unloading a direct document did not start a search")
	}
	checkSearch(t, cmds[0](), dir, path)
}
