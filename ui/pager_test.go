package ui

import "testing"

func newTestPagerModel() pagerModel {
	common := &commonModel{styles: newStyles(true)}
	m := newPagerModel(common)
	m.setSize(80, 24)
	return m
}

func TestStartSearchEntersSearchState(t *testing.T) {
	m := newTestPagerModel()
	m.viewport.SetContent("hello world\nhello again\n")

	m.startSearch()

	if m.state != pagerStateSearch {
		t.Fatalf("expected pagerStateSearch, got %v", m.state)
	}
	if !m.searchInput.Focused() {
		t.Fatal("expected search input to be focused")
	}
}

func TestConfirmSearchWithMatchesSetsSearchingState(t *testing.T) {
	m := newTestPagerModel()
	m.viewport.SetContent("hello world\nhello again\n")
	m.startSearch()
	m.searchInput.SetValue("hello")

	m.confirmSearch()

	if !m.searching {
		t.Fatal("expected searching to be true")
	}
	if m.searchCount != 2 {
		t.Fatalf("expected 2 matches, got %d", m.searchCount)
	}
	if m.state != pagerStateBrowse {
		t.Fatalf("expected to return to browse state, got %v", m.state)
	}
}

func TestConfirmSearchWithNoMatchesClearsSearching(t *testing.T) {
	m := newTestPagerModel()
	m.viewport.SetContent("hello world\n")
	m.startSearch()
	m.searchInput.SetValue("xyz")

	m.confirmSearch()

	if m.searching {
		t.Fatal("expected searching to be false when there are no matches")
	}
	if m.searchCount != 0 {
		t.Fatalf("expected 0 matches, got %d", m.searchCount)
	}
}

func TestConfirmSearchWithEmptyQueryIsNoOp(t *testing.T) {
	m := newTestPagerModel()
	m.viewport.SetContent("hello world\n")
	m.startSearch()

	m.confirmSearch()

	if m.searching {
		t.Fatal("expected searching to remain false for an empty query")
	}
	if m.state != pagerStateBrowse {
		t.Fatalf("expected to return to browse state, got %v", m.state)
	}
}

func TestCancelSearchReturnsToBrowseWithoutSearching(t *testing.T) {
	m := newTestPagerModel()
	m.viewport.SetContent("hello world\n")
	m.startSearch()
	m.searchInput.SetValue("hello")

	m.cancelSearch()

	if m.state != pagerStateBrowse {
		t.Fatalf("expected browse state, got %v", m.state)
	}
	if m.searching {
		t.Fatal("expected searching to be false after cancel")
	}
}

func TestClearSearchResetsState(t *testing.T) {
	m := newTestPagerModel()
	m.viewport.SetContent("hello world\nhello again\n")
	m.startSearch()
	m.searchInput.SetValue("hello")
	m.confirmSearch()

	m.clearSearch()

	if m.searching || m.searchQuery != "" || m.searchCount != 0 {
		t.Fatalf("expected search state fully reset, got searching=%v query=%q count=%d",
			m.searching, m.searchQuery, m.searchCount)
	}
}

func TestReapplySearchRecomputesMatchesAfterContentChanges(t *testing.T) {
	m := newTestPagerModel()
	m.viewport.SetContent("hello world\n")
	m.startSearch()
	m.searchInput.SetValue("hello")
	m.confirmSearch()

	m.viewport.SetContent("hello world\nhello again\nhello once more\n")
	m.reapplySearch()

	if m.searchCount != 3 {
		t.Fatalf("expected 3 matches after reapply, got %d", m.searchCount)
	}
}

func TestReapplySearchClearsSearchingWhenNoLongerMatching(t *testing.T) {
	m := newTestPagerModel()
	m.viewport.SetContent("hello world\n")
	m.startSearch()
	m.searchInput.SetValue("hello")
	m.confirmSearch()

	m.viewport.SetContent("goodbye world\n")
	m.reapplySearch()

	if m.searching {
		t.Fatal("expected searching to become false when content no longer matches")
	}
}

func TestUnloadClearsSearchState(t *testing.T) {
	m := newTestPagerModel()
	m.viewport.SetContent("hello world\n")
	m.startSearch()
	m.searchInput.SetValue("hello")
	m.confirmSearch()

	m.unload()

	if m.searching || m.state != pagerStateBrowse {
		t.Fatalf("expected search state cleared after unload, got searching=%v state=%v", m.searching, m.state)
	}
}

func TestNextAndPreviousMatchAreNoOpsWhenNotSearching(t *testing.T) {
	m := newTestPagerModel()
	m.viewport.SetContent("hello world\n")

	// Must not panic even though there are no highlights set and no
	// search is active.
	m.nextMatch()
	m.previousMatch()

	if m.searching {
		t.Fatal("expected searching to remain false")
	}
}

func TestNextAndPreviousMatchWorkWhileSearching(t *testing.T) {
	m := newTestPagerModel()
	m.viewport.SetContent("hello world\nhello again\n")
	m.startSearch()
	m.searchInput.SetValue("hello")
	m.confirmSearch()

	// Must not panic when cycling through matches, including wrapping
	// around past the last/first match.
	m.nextMatch()
	m.nextMatch()
	m.nextMatch()
	m.previousMatch()
	m.previousMatch()
	m.previousMatch()

	if !m.searching {
		t.Fatal("expected searching to remain true")
	}
}
