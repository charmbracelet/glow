package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestNewModelLoadsDocumentBody(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte("# Hello\n\nSome text here.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModel(Config{Path: path}, "").(model)
	if m.state != stateShowDocument {
		t.Fatalf("expected stateShowDocument, got %s", m.state)
	}
	if !strings.Contains(m.pager.currentDocument.Body, "Some text") {
		t.Errorf("expected document body to be loaded, got %q", m.pager.currentDocument.Body)
	}
}

func TestResizeKeepsDocumentContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte("# Hello\n\nSome text here.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModel(Config{Path: path}, "").(model)
	m.pager.setSize(80, 24)

	// A resize re-renders the document from its body.
	pager, cmd := m.pager.update(tea.WindowSizeMsg{Width: 60, Height: 24})
	if cmd == nil {
		t.Fatal("expected a render command after resize")
	}
	msg := cmd()
	rendered, ok := msg.(contentRenderedMsg)
	if !ok {
		t.Fatalf("expected contentRenderedMsg, got %T", msg)
	}
	pager, _ = pager.update(rendered)

	if !strings.Contains(pager.View(), "Some text") {
		t.Errorf("expected document text after resize, got %q", pager.View())
	}
}
