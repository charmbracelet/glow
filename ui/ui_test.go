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

func TestSearchInputTakesPriorityOverGlobalShortcuts(t *testing.T) {
	tests := []struct {
		name           string
		key            tea.KeyPressMsg
		wantPagerState pagerState
		wantInput      string
	}{
		{name: "h", key: tea.KeyPressMsg{Code: 'h', Text: "h"}, wantPagerState: pagerStateSearch, wantInput: "h"},
		{name: "q", key: tea.KeyPressMsg{Code: 'q', Text: "q"}, wantPagerState: pagerStateSearch, wantInput: "q"},
		{name: "left", key: tea.KeyPressMsg{Code: tea.KeyLeft}, wantPagerState: pagerStateSearch},
		{name: "delete", key: tea.KeyPressMsg{Code: tea.KeyDelete}, wantPagerState: pagerStateSearch},
		{name: "escape", key: tea.KeyPressMsg{Code: tea.KeyEscape}, wantPagerState: pagerStateBrowse},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newModel(Config{}, "# Test document").(model)
			updated, _ := m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
			m = updated.(model)

			updated, _ = m.Update(tc.key)
			m = updated.(model)

			if m.state != stateShowDocument {
				t.Fatalf("expected document to remain open, got %s", m.state)
			}
			if m.pager.state != tc.wantPagerState {
				t.Fatalf("expected pager state %d, got %d", tc.wantPagerState, m.pager.state)
			}
			if got := m.pager.searchInput.Value(); got != tc.wantInput {
				t.Errorf("expected search input %q, got %q", tc.wantInput, got)
			}
		})
	}
}
