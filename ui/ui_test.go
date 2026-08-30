package ui

import (
	tea "charm.land/bubbletea/v2"
	"testing"
)

// newTestModel builds a model already sitting on a loaded document, bypassing
// the async load/render commands that Init() would normally kick off. This
// lets tests drive the real top-level model.Update routing layer (the layer
// where the esc-during-active-search bug lived) instead of only exercising
// pagerModel methods directly.
func newTestModel(content string) model {
	common := &commonModel{styles: newStyles(true)}
	m := model{
		common: common,
		state:  stateShowDocument,
		pager:  newPagerModel(common),
		stash:  newStashModel(common),
	}
	m.pager.setSize(80, 24)
	m.pager.setContent(content)
	return m
}

// pressKey builds a tea.KeyPressMsg for a single printable character.
func pressKey(s string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Text: s, Code: rune(s[0])}
}

// pressCode builds a tea.KeyPressMsg for a named key (e.g. tea.KeyEscape).
func pressCode(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

func TestModelUpdateSearchLifecycleThroughRouting(t *testing.T) {
	m := newTestModel("hello world\nhello again\n")

	// Enter search mode via "/".
	updated, _ := m.Update(pressKey("/"))
	mm, ok := updated.(model)
	if !ok {
		t.Fatalf("expected model, got %T", updated)
	}
	if mm.pager.state != pagerStateSearch {
		t.Fatalf("expected pagerStateSearch after '/', got %v", mm.pager.state)
	}

	// Type "hello" as individual key presses through the real routing layer.
	for _, ch := range "hello" {
		updated, _ = mm.Update(pressKey(string(ch)))
		mm = updated.(model)
	}
	if mm.pager.searchInput.Value() != "hello" {
		t.Fatalf("expected search input value %q, got %q", "hello", mm.pager.searchInput.Value())
	}

	// Confirm the search with enter.
	updated, _ = mm.Update(pressCode(tea.KeyEnter))
	mm = updated.(model)
	if !mm.pager.searching {
		t.Fatal("expected an active/confirmed search after enter")
	}
	if len(mm.pager.searchMatches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(mm.pager.searchMatches))
	}
	if mm.state != stateShowDocument {
		t.Fatalf("expected to remain in stateShowDocument after confirming search, got %v", mm.state)
	}

	// n/N should route to the pager and cycle matches without quitting or
	// unloading the document.
	updated, cmd := mm.Update(pressKey("n"))
	mm = updated.(model)
	if cmd != nil {
		if _, isQuit := cmd().(tea.QuitMsg); isQuit {
			t.Fatal("did not expect 'n' to quit the program")
		}
	}
	if mm.state != stateShowDocument || !mm.pager.searching {
		t.Fatalf("expected 'n' to leave state/searching unaffected, got state=%v searching=%v", mm.state, mm.pager.searching)
	}

	updated, cmd = mm.Update(pressKey("N"))
	mm = updated.(model)
	if cmd != nil {
		if _, isQuit := cmd().(tea.QuitMsg); isQuit {
			t.Fatal("did not expect 'N' to quit the program")
		}
	}
	if mm.state != stateShowDocument || !mm.pager.searching {
		t.Fatalf("expected 'N' to leave state/searching unaffected, got state=%v searching=%v", mm.state, mm.pager.searching)
	}

	// esc with an active/confirmed search must clear the search but keep the
	// document open -- this is the exact case the critical bug broke, where
	// esc fell through to unloadDocument() and kicked the user back to the
	// file listing.
	updated, _ = mm.Update(pressCode(tea.KeyEscape))
	mm = updated.(model)
	if mm.pager.searching {
		t.Fatal("expected searching to be false after esc clears an active search")
	}
	if mm.state != stateShowDocument {
		t.Fatalf("expected esc to keep the document open (stateShowDocument), got %v", mm.state)
	}
}

func TestModelUpdateEscWhileTypingQueryStillCancels(t *testing.T) {
	m := newTestModel("hello world\n")

	updated, _ := m.Update(pressKey("/"))
	mm := updated.(model)

	updated, _ = mm.Update(pressKey("h"))
	mm = updated.(model)

	updated, _ = mm.Update(pressCode(tea.KeyEscape))
	mm = updated.(model)

	if mm.pager.state != pagerStateBrowse {
		t.Fatalf("expected browse state after esc while typing, got %v", mm.pager.state)
	}
	if mm.pager.searching {
		t.Fatal("expected searching to remain false after cancelling an unconfirmed query")
	}
	if mm.state != stateShowDocument {
		t.Fatalf("expected to remain in stateShowDocument, got %v", mm.state)
	}
}

func TestModelUpdateEscWithNoSearchStillUnloadsDocument(t *testing.T) {
	m := newTestModel("hello world\n")

	updated, _ := m.Update(pressCode(tea.KeyEscape))
	mm := updated.(model)

	if mm.state != stateShowStash {
		t.Fatalf("expected esc with no active search to unload the document (stateShowStash), got %v", mm.state)
	}
}

func TestModelUpdateQTypedWhileSearchInputFocusedDoesNotQuit(t *testing.T) {
	m := newTestModel("hello world\n")

	updated, _ := m.Update(pressKey("/"))
	mm := updated.(model)

	updated, cmd := mm.Update(pressKey("q"))
	mm = updated.(model)

	if cmd != nil {
		if _, isQuit := cmd().(tea.QuitMsg); isQuit {
			t.Fatal("did not expect 'q' typed into the search input to quit the program")
		}
	}
	if mm.pager.state != pagerStateSearch {
		t.Fatalf("expected to remain in pagerStateSearch, got %v", mm.pager.state)
	}
	if mm.pager.searchInput.Value() != "q" {
		t.Fatalf("expected 'q' to be inserted into the search input, got %q", mm.pager.searchInput.Value())
	}
}
