package ui

import (
	"strings"
	"testing"
)

func newTestPagerModel() pagerModel {
	common := &commonModel{styles: newStyles(true), width: 80, height: 24}
	m := newPagerModel(common)
	m.setSize(80, 24)
	return m
}

// TestGlamourRenderTextSizing must run before any other test in this package
// that calls glamourRender (e.g. TestSearchHighlightsAreVisibleInRealGlamourRenderedContent
// below): utils.TextSizingEnabled() caches its result for the lifetime of the
// test binary via sync.OnceValue, so whichever test calls glamourRender first
// permanently determines that value for every subsequent test. This test
// relies on overriding it to "on" via GLOW_TEXT_SIZING, so it's declared
// first to win that race deterministically (Go runs tests within a file in
// declaration order). Content without headings, like the other glamourRender
// callers in this file use, is unaffected either way -- AddHeadingSizeMarkers
// only touches heading styling.
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

func TestStartSearchEntersSearchState(t *testing.T) {
	m := newTestPagerModel()
	m.setContent("hello world\nhello again\n")

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
	m.setContent("hello world\nhello again\n")
	m.startSearch()
	m.searchInput.SetValue("hello")

	m.confirmSearch()

	if !m.searching {
		t.Fatal("expected searching to be true")
	}
	if len(m.searchMatches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(m.searchMatches))
	}
	if m.state != pagerStateBrowse {
		t.Fatalf("expected to return to browse state, got %v", m.state)
	}
}

func TestConfirmSearchWithNoMatchesClearsSearching(t *testing.T) {
	m := newTestPagerModel()
	m.setContent("hello world\n")
	m.startSearch()
	m.searchInput.SetValue("xyz")

	m.confirmSearch()

	if m.searching {
		t.Fatal("expected searching to be false when there are no matches")
	}
	if len(m.searchMatches) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(m.searchMatches))
	}
}

func TestConfirmSearchWithEmptyQueryIsNoOp(t *testing.T) {
	m := newTestPagerModel()
	m.setContent("hello world\n")
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
	m.setContent("hello world\n")
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
	m.setContent("hello world\nhello again\n")
	m.startSearch()
	m.searchInput.SetValue("hello")
	m.confirmSearch()

	m.clearSearch()

	if m.searching || m.searchQuery != "" || len(m.searchMatches) != 0 {
		t.Fatalf("expected search state fully reset, got searching=%v query=%q count=%d",
			m.searching, m.searchQuery, len(m.searchMatches))
	}
}

func TestReapplySearchRecomputesMatchesAfterContentChanges(t *testing.T) {
	m := newTestPagerModel()
	m.setContent("hello world\n")
	m.startSearch()
	m.searchInput.SetValue("hello")
	m.confirmSearch()

	m.setContent("hello world\nhello again\nhello once more\n")
	m.reapplySearch()

	if len(m.searchMatches) != 3 {
		t.Fatalf("expected 3 matches after reapply, got %d", len(m.searchMatches))
	}
}

func TestReapplySearchClearsSearchingWhenNoLongerMatching(t *testing.T) {
	m := newTestPagerModel()
	m.setContent("hello world\n")
	m.startSearch()
	m.searchInput.SetValue("hello")
	m.confirmSearch()

	m.setContent("goodbye world\n")
	m.reapplySearch()

	if m.searching {
		t.Fatal("expected searching to become false when content no longer matches")
	}
}

func TestUnloadClearsSearchState(t *testing.T) {
	m := newTestPagerModel()
	m.setContent("hello world\n")
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
	m.setContent("hello world\n")

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
	m.setContent("hello world\nhello again\n")
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

// TestSearchHighlightsAreVisibleInRealGlamourRenderedContent is the
// regression test for the original bug: viewport.Model.SetHighlights
// silently misattributes matches to the wrong line/column once the content
// contains ANSI escape codes, which is always true for glamour-rendered
// markdown. It's not enough to assert that matches are found -- we must
// confirm the baked-in highlight styling actually wraps the matched word in
// the rendered view.
func TestSearchHighlightsAreVisibleInRealGlamourRenderedContent(t *testing.T) {
	common := &commonModel{
		styles: newStyles(true),
		cfg: Config{
			GlamourEnabled:  true,
			GlamourMaxWidth: 80,
			GlamourStyle:    "dark",
		},
	}
	m := newPagerModel(common)
	m.setSize(80, 24)
	m.currentDocument = markdown{Note: "test.md"}

	md := "Some padding text so the target word is not at the very start of the document.\n\n" +
		"Here is the special target word we will search for.\n"

	rendered, err := glamourRender(m, md)
	if err != nil {
		t.Fatalf("glamourRender: %v", err)
	}
	if !strings.Contains(rendered, "\x1b[") {
		t.Fatal("expected glamour-rendered content to contain ANSI escape codes (sanity check for this test)")
	}

	m.setContent(rendered)
	m.startSearch()
	m.searchInput.SetValue("target")
	m.confirmSearch()

	if len(m.searchMatches) != 2 {
		t.Fatalf("expected 2 matches for 'target', got %d", len(m.searchMatches))
	}

	view := m.viewport.View()

	selectedANSI := m.common.styles.searchSelectedHighlightStyle.Render("target")
	if !strings.Contains(view, selectedANSI) {
		t.Fatalf("expected viewport view to contain the selected-match highlight ANSI wrapping %q, got:\n%s", "target", view)
	}
}

func TestStatusBarShowsSearchQueryAndMatchCount(t *testing.T) {
	m := newTestPagerModel()
	m.setContent("hello world\nhello again\nhello once more\n")
	m.startSearch()
	m.searchInput.SetValue("hello")
	m.confirmSearch()

	var b strings.Builder
	m.statusBarView(&b)
	bar := b.String()

	if !strings.Contains(bar, "Search: hello") {
		t.Fatalf("expected status bar to contain the search query, got: %q", bar)
	}
	if !strings.Contains(bar, "1/3 matches") {
		t.Fatalf("expected status bar to show the current match position out of the total, got: %q", bar)
	}
}

func TestStatusBarMatchPositionAdvancesWithNextMatch(t *testing.T) {
	m := newTestPagerModel()
	m.setContent("hello world\nhello again\nhello once more\n")
	m.startSearch()
	m.searchInput.SetValue("hello")
	m.confirmSearch()

	m.nextMatch()

	var b strings.Builder
	m.statusBarView(&b)
	bar := b.String()

	if !strings.Contains(bar, "2/3 matches") {
		t.Fatalf("expected status bar to advance to the 2nd of 3 matches after nextMatch, got: %q", bar)
	}
}

func TestStatusBarTruncatesLongSearchQuery(t *testing.T) {
	m := newTestPagerModel()
	longQuery := strings.Repeat("a", 100)
	content := longQuery + "\n"
	m.setContent(content)
	m.startSearch()
	m.searchInput.SetValue(longQuery)
	m.confirmSearch()

	var b strings.Builder
	m.statusBarView(&b)
	bar := b.String()

	if strings.Contains(bar, longQuery) {
		t.Fatalf("expected the full 100-char query not to appear verbatim in the status bar, got: %q", bar)
	}
	if !strings.Contains(bar, ellipsis) {
		t.Fatalf("expected a truncated query to end with the ellipsis, got: %q", bar)
	}
	if !strings.Contains(bar, "1 match") {
		t.Fatalf("expected the match count to still be visible alongside the truncated query, got: %q", bar)
	}
}
