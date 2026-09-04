package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func newTestModel(t *testing.T, documentOnly bool) model {
	t.Helper()

	m := model{
		common: &commonModel{
			styles:       newStyles(true),
			width:        20,
			height:       10,
			documentOnly: documentOnly,
		},
		state: stateShowDocument,
	}
	m.pager = newPagerModel(m.common)
	m.stash = newStashModel(m.common)
	m.pager.setSize(m.common.width, m.common.height)
	m.pager.setContent(strings.Repeat("x", 200))
	return m
}

func press(t *testing.T, m model, key rune) model {
	t.Helper()

	newModel, _ := m.Update(tea.KeyPressMsg{Code: key, Text: string(key)})
	return newModel.(model)
}

func TestScrollLeftBeforeLeavingDocument(t *testing.T) {
	m := newTestModel(t, false)

	m = press(t, m, 'l')
	offset := m.pager.viewport.XOffset()
	if offset == 0 {
		t.Fatal("l did not scroll right")
	}

	m = press(t, m, 'h')
	if m.state != stateShowDocument {
		t.Fatal("h left the document while it was scrolled right")
	}
	if m.pager.viewport.XOffset() >= offset {
		t.Fatalf("h did not scroll left: %d -> %d", offset, m.pager.viewport.XOffset())
	}

	for i := 0; i < 20 && m.pager.viewport.XOffset() > 0; i++ {
		m = press(t, m, 'h')
	}

	m = press(t, m, 'h')
	if m.state != stateShowStash {
		t.Fatalf("h at the left edge did not go back to the file listing, state = %v", m.state)
	}
}

func TestDocumentOnlyStaysInDocument(t *testing.T) {
	m := newTestModel(t, true)

	m = press(t, m, 'h')
	if m.state != stateShowDocument {
		t.Fatal("h left the only document there is")
	}

	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = newModel.(model)
	if m.state != stateShowDocument {
		t.Fatal("esc left the only document there is")
	}
}
