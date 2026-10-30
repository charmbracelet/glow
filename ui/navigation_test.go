package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestStashControlKeysMoveSelection(t *testing.T) {
	common := &commonModel{
		styles: newStyles(true),
		width:  80,
		height: 24,
	}
	m := newStashModel(common)
	m.markdowns = []*markdown{
		{Note: "first.md"},
		{Note: "second.md"},
		{Note: "third.md"},
	}
	m.setSize(common.width, common.height)
	m.setCursor(1)

	m.handleDocumentBrowsing(controlKey('n'))
	if got := m.cursor(); got != 2 {
		t.Fatalf("expected ctrl+n to move selection down to 2, got %d", got)
	}

	m.handleDocumentBrowsing(controlKey('p'))
	if got := m.cursor(); got != 1 {
		t.Fatalf("expected ctrl+p to move selection up to 1, got %d", got)
	}
}

func TestControlNavigationDoesNotOverrideFilterInput(t *testing.T) {
	common := &commonModel{
		styles: newStyles(true),
		width:  80,
		height: 24,
	}
	m := newStashModel(common)
	m.markdowns = []*markdown{{Note: "first.md"}, {Note: "second.md"}}
	m.setSize(common.width, common.height)
	m.setCursor(1)
	m.filterState = filtering
	m.filterInput.SetValue("first")

	m, _ = m.update(controlKey('n'))
	if got := m.cursor(); got != 1 {
		t.Fatalf("expected ctrl+n to leave selection unchanged while filtering, got %d", got)
	}
	if got := m.filterInput.Value(); got != "first" {
		t.Fatalf("expected filter input to remain unchanged, got %q", got)
	}
}

func TestControlNavigationDoesNotScrollDocument(t *testing.T) {
	common := &commonModel{
		styles: newStyles(true),
		width:  80,
		height: 10,
	}
	m := newPagerModel(common)
	m.setSize(common.width, common.height)
	m.viewport.SetContent(strings.Repeat("line\n", 30))
	m.viewport.SetYOffset(5)

	m, _ = m.update(controlKey('n'))
	if got := m.viewport.YOffset(); got != 5 {
		t.Fatalf("expected ctrl+n not to scroll document details, got offset %d", got)
	}

	m, _ = m.update(controlKey('p'))
	if got := m.viewport.YOffset(); got != 5 {
		t.Fatalf("expected ctrl+p not to scroll document details, got offset %d", got)
	}
}

func controlKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Mod: tea.ModCtrl}
}
