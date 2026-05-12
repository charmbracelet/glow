package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// drainCmd runs a tea.Cmd to completion and returns every concrete message it
// produces, recursing into tea.BatchMsg. Long-blocking subscription commands
// (e.g. spinner ticks, fsnotify) are skipped via a short deadline so tests
// don't hang.
func drainCmd(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()
	if cmd == nil {
		return nil
	}
	out := make(chan tea.Msg, 1)
	go func() {
		defer func() { _ = recover() }()
		out <- cmd()
	}()
	select {
	case msg := <-out:
		if msg == nil {
			return nil
		}
		if batch, ok := msg.(tea.BatchMsg); ok {
			var msgs []tea.Msg
			for _, c := range batch {
				msgs = append(msgs, drainCmd(t, c)...)
			}
			return msgs
		}
		return []tea.Msg{msg}
	case <-time.After(200 * time.Millisecond):
		return nil
	}
}

// runUpdate feeds a single message into Update and drains every resulting
// command, cascading the produced messages back through Update. Returns the
// final model after the cascade settles.
func runUpdate(t *testing.T, m tea.Model, msg tea.Msg) tea.Model {
	t.Helper()
	mm, cmd := m.Update(msg)
	for _, follow := range drainCmd(t, cmd) {
		mm = runUpdate(t, mm, follow)
	}
	return mm
}

// TestResizeReflowsContent confirms that, after a WindowSizeMsg, the pager
// re-renders the current document at the new width rather than leaving the
// viewport empty. Regression test for the SIGWINCH/resize bug where
// currentDocument.Body was never populated for files opened by path.
func TestResizeReflowsContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	body := "# Hello\n\nThis is a paragraph with enough words to wrap nicely across multiple lines so we can observe reflow at different widths.\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Path:            path,
		GlamourEnabled:  true,
		GlamourStyle:    "notty",
		GlamourMaxWidth: 120,
	}
	config = cfg

	m := newModel(cfg, "")
	for _, msg := range drainCmd(t, m.Init()) {
		m = runUpdate(t, m, msg)
	}

	mm := m.(model)
	if mm.pager.currentDocument.Body == "" {
		t.Fatalf("after Init, currentDocument.Body should be populated; got empty")
	}
	if !strings.Contains(mm.pager.currentDocument.Body, "Hello") {
		t.Fatalf("currentDocument.Body missing source content; got %q", mm.pager.currentDocument.Body)
	}

	m = runUpdate(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	mm = m.(model)
	narrowView := mm.View()
	if narrowView == "" || !strings.Contains(narrowView, "Hello") {
		t.Fatalf("after WindowSizeMsg{80,24}, view should contain rendered content; got %q", narrowView)
	}

	m = runUpdate(t, m, tea.WindowSizeMsg{Width: 120, Height: 24})
	mm = m.(model)
	wideView := mm.View()
	if wideView == "" || !strings.Contains(wideView, "Hello") {
		t.Fatalf("after WindowSizeMsg{120,24}, view should contain rendered content; got %q", wideView)
	}

	if narrowView == wideView {
		t.Fatalf("view unchanged across resize; resize did not reflow content")
	}
}

// TestResizeStripsFrontmatterOnce confirms that loaded documents have their
// frontmatter stripped exactly once and stored on currentDocument.Body, so
// resize re-renders don't expose the frontmatter.
func TestResizeStripsFrontmatterOnce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	body := "---\ntitle: Test\n---\n\n# Hello\n\nBody text.\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Path:            path,
		GlamourEnabled:  true,
		GlamourStyle:    "notty",
		GlamourMaxWidth: 120,
	}
	config = cfg

	m := newModel(cfg, "")
	for _, msg := range drainCmd(t, m.Init()) {
		m = runUpdate(t, m, msg)
	}
	mm := m.(model)

	if strings.Contains(mm.pager.currentDocument.Body, "title: Test") {
		t.Fatalf("currentDocument.Body still contains frontmatter: %q", mm.pager.currentDocument.Body)
	}
	if !strings.Contains(mm.pager.currentDocument.Body, "Hello") {
		t.Fatalf("currentDocument.Body missing body content; got %q", mm.pager.currentDocument.Body)
	}
}
