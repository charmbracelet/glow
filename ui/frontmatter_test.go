package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestDocumentFrontmatter(t *testing.T) {
	const document = "---\nname: first-config\n---\nname: second-config\n"
	for _, tc := range []struct {
		filename string
		want     string
	}{
		{filename: "manifests.yaml", want: document},
		{filename: "manifests.yml", want: document},
		{filename: "notes.txt", want: document},
		{filename: "article.md", want: "name: second-config\n"},
		{filename: "README", want: "name: second-config\n"},
	} {
		t.Run(tc.filename, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tc.filename)
			if err := os.WriteFile(path, []byte(document), 0o600); err != nil {
				t.Fatal(err)
			}

			m := newModel(Config{Path: path}, "").(model)
			if m.pager.watcher != nil {
				t.Cleanup(func() { _ = m.pager.watcher.Close() })
			}
			if m.fatalErr != nil {
				t.Fatal(m.fatalErr)
			}
			if got := m.pager.currentDocument.Body; got != tc.want {
				t.Errorf("initial body = %q, want %q", got, tc.want)
			}

			// Loading from the file browser and reloading use the fetched message
			// path rather than the initial-file path.
			_, cmd := m.Update(fetchedMarkdownMsg(&markdown{
				localPath: path,
				Note:      tc.filename,
				Body:      document,
			}))
			var rendered bool
			var run func(tea.Cmd)
			run = func(cmd tea.Cmd) {
				if cmd == nil {
					return
				}
				switch msg := cmd().(type) {
				case tea.BatchMsg:
					for _, child := range msg {
						run(child)
					}
				case contentRenderedMsg:
					rendered = true
					if string(msg) != tc.want {
						t.Errorf("reloaded body = %q, want %q", msg, tc.want)
					}
				case errMsg:
					t.Fatal(msg)
				}
			}
			run(cmd)
			if !rendered {
				t.Fatal("no rendered document returned")
			}
		})
	}
}
