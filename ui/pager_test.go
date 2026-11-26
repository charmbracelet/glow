package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/fsnotify/fsnotify"
)

func TestGlamourRenderTextSizing(t *testing.T) {
	t.Setenv("GLOW_TEXT_SIZING", "on")

	common := &commonModel{
		cfg: Config{
			GlamourEnabled:  true,
			GlamourStyle:    "dark",
			GlamourMaxWidth: 80,
		},
		styles: newStyles(true),
		width:  80,
		height: 24,
	}
	m := newPagerModel(common)
	m.currentDocument = markdown{Note: "test.md"}
	m.setSize(80, 24)

	out, err := glamourRender(m, "# Hello\n\n## World\n")
	if err != nil {
		t.Fatal(err)
	}
	// The dark style's H1 has a background color, which renders broken when
	// scaled, so it is left at normal size.
	if strings.Contains(out, "\x1b]66;s=3;") {
		t.Errorf("expected H1 with background to not be scaled, got %q", out)
	}
	if !strings.Contains(out, "Hello") {
		t.Errorf("expected H1 text in pager output, got %q", out)
	}
	if !strings.Contains(out, "\x1b]66;s=2;World\x1b\\") {
		t.Errorf("expected H2 at 2x scale in pager output, got %q", out)
	}
	if strings.Contains(out, "\x1b]6666;") {
		t.Errorf("expected no leftover markers in pager output, got %q", out)
	}
}

func TestWatchFileWithNoncanonicalPath(t *testing.T) {
	for _, name := range []string{"dot segment", "different case"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "test.md")
			if err := os.WriteFile(path, []byte("before"), 0o644); err != nil {
				t.Fatal(err)
			}

			sep := string(os.PathSeparator)
			documentPath := dir + sep + "." + sep + "test.md"
			if name == "different case" {
				var err error
				documentPath, err = filepath.Abs(filepath.Join(dir, "TEST.md"))
				if err != nil {
					t.Fatal(err)
				}
				if _, err := os.Stat(documentPath); os.IsNotExist(err) {
					t.Skip("file system is case-sensitive")
				} else if err != nil {
					t.Fatal(err)
				}
			}

			watcher, err := fsnotify.NewWatcher()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = watcher.Close() })
			if err := watcher.Add(dir); err != nil {
				t.Fatal(err)
			}

			done := make(chan tea.Msg, 1)
			go func() { done <- waitForFileChange(watcher, documentPath) }()

			if err := os.WriteFile(path, []byte("after"), 0o644); err != nil {
				t.Fatal(err)
			}

			select {
			case msg := <-done:
				if _, ok := msg.(reloadMsg); !ok {
					t.Fatalf("expected reload message, got %T", msg)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("editing the watched file did not trigger a reload")
			}
		})
	}
}
