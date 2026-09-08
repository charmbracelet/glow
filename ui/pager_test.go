package ui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

// newTestPager returns a pager model pointing at a markdown file in a fresh
// temporary directory, along with that file's path.
func newTestPager(t *testing.T) (*pagerModel, string) {
	t.Helper()

	dir := t.TempDir()
	// macOS hands out symlinked temp dirs (/var -> /private/var), while
	// fsnotify reports the resolved path, so resolve it up front.
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}

	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte("# hello\n"), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}

	m := newPagerModel(&commonModel{})
	m.currentDocument = markdown{localPath: path}
	return &m, path
}

// When fsnotify can't create a watcher — the usual cause being an exhausted
// inotify instance limit — the pager must degrade to "no live reload" rather
// than dereference a nil watcher.
func TestPagerWatcherInitFailure(t *testing.T) {
	orig := newWatcher
	newWatcher = func() (*fsnotify.Watcher, error) {
		return nil, errors.New("too many open files")
	}
	t.Cleanup(func() { newWatcher = orig })

	m, _ := newTestPager(t)

	if m.watcher != nil {
		t.Fatalf("expected nil watcher after init failure, got %#v", m.watcher)
	}

	// Each of these used to panic with a nil pointer dereference.
	if msg := m.watchFile(); msg != nil {
		t.Errorf("expected nil msg from watchFile with no watcher, got %#v", msg)
	}
	m.unwatchFile()
	m.unload()
}

// The watcher is still nil-safe if init failed and the document changes, which
// is the path the pager takes on every render.
func TestPagerWatchFileNilWatcherRepeatedly(t *testing.T) {
	m, _ := newTestPager(t)
	m.watcher = nil

	for range 3 {
		if msg := m.watchFile(); msg != nil {
			t.Fatalf("expected nil msg from watchFile with no watcher, got %#v", msg)
		}
		m.unwatchFile()
	}
}

// A working watcher must still report writes to the current document.
func TestPagerWatchFileReload(t *testing.T) {
	m, path := newTestPager(t)
	if m.watcher == nil {
		t.Skip("could not create fsnotify watcher on this system")
	}
	t.Cleanup(func() { _ = m.watcher.Close() })

	msgs := make(chan any, 1)
	go func() { msgs <- m.watchFile() }()

	// Give watchFile a moment to register the directory before touching it.
	time.Sleep(100 * time.Millisecond)
	if err := os.WriteFile(path, []byte("# hello again\n"), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}

	select {
	case msg := <-msgs:
		if _, ok := msg.(reloadMsg); !ok {
			t.Fatalf("expected reloadMsg, got %#v", msg)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for reloadMsg")
	}
}
