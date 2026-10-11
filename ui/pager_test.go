package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

// TestReloadMsgDoesNotClearActiveSearch is the regression test for a
// reviewer suggestion: a reload triggered by the file watcher shouldn't
// discard the user's in-progress search. contentRenderedMsg's own
// m.reapplySearch() call re-runs it once the reloaded content actually
// renders (see TestContentRenderedMsgReappliesActiveSearchAfterReload
// below), so reloadMsg itself just needs to leave the search state alone.
func TestReloadMsgDoesNotClearActiveSearch(t *testing.T) {
	m := newTestPagerModel()
	m.setContent("hello world\n")
	m.startSearch()
	m.searchInput.SetValue("hello")
	m.confirmSearch()

	updated, _ := m.update(reloadMsg{})

	if !updated.searching {
		t.Fatal("expected an active search to survive a reloadMsg")
	}
	if updated.searchQuery != "hello" {
		t.Fatalf("expected the search query to be preserved across reload, got %q", updated.searchQuery)
	}
}

// TestEditorFinishedMsgDoesNotClearActiveSearch mirrors
// TestReloadMsgDoesNotClearActiveSearch for the other reload path: returning
// from $EDITOR.
func TestEditorFinishedMsgDoesNotClearActiveSearch(t *testing.T) {
	m := newTestPagerModel()
	m.setContent("hello world\n")
	m.startSearch()
	m.searchInput.SetValue("hello")
	m.confirmSearch()

	updated, _ := m.update(editorFinishedMsg{})

	if !updated.searching {
		t.Fatal("expected an active search to survive returning from the editor")
	}
}

// TestContentRenderedMsgReappliesActiveSearchAfterReload confirms the other
// half of the reload-preserves-search behavior: once the reloaded content
// actually renders, the preserved query is re-run against it (rather than
// just sitting there pointing at stale, now-incorrect match positions).
func TestContentRenderedMsgReappliesActiveSearchAfterReload(t *testing.T) {
	m := newTestPagerModel()
	m.setContent("hello world\n")
	m.startSearch()
	m.searchInput.SetValue("hello")
	m.confirmSearch()

	updated, _ := m.update(contentRenderedMsg("hello again\nhello once more\n"))

	if !updated.searching {
		t.Fatal("expected the search to remain active after the reload re-renders")
	}
	if len(updated.searchMatches) != 2 {
		t.Fatalf("expected the search to be re-run against the new content, got %d matches", len(updated.searchMatches))
	}
}

// TestStatusMessageTimeoutDoesNotExitSearchPrompt is the regression test for
// a reviewer-reported bug: statusMessageTimeoutMsg unconditionally reset the
// pager to browse state, even if the user had since started typing a search
// query (e.g. after a "No matches" message's timer fires mid-typing). That
// silently closed the search prompt, after which keys like "q"/"j"/"k"
// reached the pager's normal keybindings instead of the search input.
func TestStatusMessageTimeoutDoesNotExitSearchPrompt(t *testing.T) {
	m := newTestPagerModel()
	m.setContent("hello world\n")
	m.startSearch()

	updated, _ := m.update(statusMessageTimeoutMsg(pagerContext))

	if updated.state != pagerStateSearch {
		t.Fatalf("expected to remain in pagerStateSearch, got %v", updated.state)
	}
}

// TestStatusMessageTimeoutStillExitsStatusMessage confirms the fix above
// didn't just disable statusMessageTimeoutMsg's handling outright: it
// should still return to browse state from an actual status message.
func TestStatusMessageTimeoutStillExitsStatusMessage(t *testing.T) {
	m := newTestPagerModel()
	m.setContent("hello world\n")
	m.state = pagerStateStatusMessage

	updated, _ := m.update(statusMessageTimeoutMsg(pagerContext))

	if updated.state != pagerStateBrowse {
		t.Fatalf("expected to return to pagerStateBrowse, got %v", updated.state)
	}
}

// TestPasteMsgReachesSearchInput is the regression test for a reviewer-
// reported bug: only tea.KeyPressMsg was forwarded to the search input
// while typing, so bracketed paste (which arrives as tea.PasteMsg) was
// silently dropped.
func TestPasteMsgReachesSearchInput(t *testing.T) {
	m := newTestPagerModel()
	m.setContent("hello world\n")
	m.startSearch()

	updated, _ := m.update(tea.PasteMsg{Content: "pasted text"})

	if updated.searchInput.Value() != "pasted text" {
		t.Fatalf("expected pasted text in the search input, got %q", updated.searchInput.Value())
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

// TestSearchHighlightDoesNotCorruptLineOnGraphemeClusterBoundary is the
// end-to-end regression test for the critical bug found in review: a match
// ending mid-grapheme-cluster (e.g. right after the "e" in NFD-normalized
// "café", before the combining accent) used to produce an invalid
// (colEnd < colStart) range. Fed to lipgloss.StyleRanges, that didn't just
// fail to highlight -- StyleRanges unconditionally sets its internal
// lastIdx to the invalid (too-small) End, so its final TruncateLeft call
// re-emitted content already written, duplicating the entire line in the
// rendered output.
func TestSearchHighlightDoesNotCorruptLineOnGraphemeClusterBoundary(t *testing.T) {
	m := newTestPagerModel()
	m.setContent("see café now\nsecond line here\n")
	m.startSearch()
	m.searchInput.SetValue("cafe")
	m.confirmSearch()

	view := m.viewport.View()

	if n := strings.Count(view, "now"); n != 1 {
		t.Fatalf("expected the line to appear exactly once (not duplicated), found %d occurrences of \"now\" in:\n%s", n, view)
	}
	if n := strings.Count(view, "second line here"); n != 1 {
		t.Fatalf("expected the following line to appear exactly once, found %d occurrences in:\n%s", n, view)
	}
}

// TestSearchFindsAndHighlightsTextSizedHeading is the end-to-end regression
// test for the heading/text-sizing bug reported on the PR: with kitty text
// sizing enabled, heading text renders inside OSC 66 sequences, which both
// xansi.Strip (used for matching) and lipgloss.StyleRanges' internal
// stripping (used for highlighting) mishandle, since neither understands
// that protocol's payload is visible text. This exercises the full
// startSearch -> confirmSearch -> applyHighlights path against a real
// glamour-rendered heading, rather than just the lower-level findMatches
// unit tests.
func TestSearchFindsAndHighlightsTextSizedHeading(t *testing.T) {
	t.Setenv("GLOW_TEXT_SIZING", "on")

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

	rendered, err := glamourRender(m, "## Installation\n\nSome body text.\n")
	if err != nil {
		t.Fatalf("glamourRender: %v", err)
	}
	if !strings.Contains(rendered, "\x1b]66;") {
		t.Fatal("expected text-sizing OSC 66 sequences in rendered output (sanity check)")
	}

	m.setContent(rendered)
	m.startSearch()
	m.searchInput.SetValue("installation")
	m.confirmSearch()

	if len(m.searchMatches) != 1 {
		t.Fatalf("expected 1 match inside the heading, got %d", len(m.searchMatches))
	}

	view := m.viewport.View()

	// The whole matched word should be re-wrapped as a single OSC 66
	// sequence (preserving the "s=2" size) with the selected-highlight style
	// applied around it -- not one OSC 66 sequence per character, which
	// would still render correctly but needlessly multiply escape-sequence
	// overhead for longer matches.
	wantFragment := m.common.styles.searchSelectedHighlightStyle.Render(
		"\x1b]66;s=2;Installation\x1b\\",
	)
	if !strings.Contains(view, wantFragment) {
		t.Fatalf("expected view to contain the heading word highlighted as a single size-preserved run %q, got:\n%s",
			wantFragment, view)
	}
}

// TestHeadingStylePreservedAfterHighlightedMatch is the regression test for
// a reviewer-reported bug: text-sized headings originally emit only one
// color/bold SGR code before a whole chunk of OSC 66-wrapped text (e.g.
// "Hello world" as a single chunk). Splitting that chunk into a highlighted
// sub-run ("Hello") and a plain one (" world") meant the plain sub-run,
// coming right after a highlight style's Render() call (which always ends
// in a reset), lost the heading's styling entirely -- " world" rendered as
// plain unstyled text instead of bold/colored.
func TestHeadingStylePreservedAfterHighlightedMatch(t *testing.T) {
	t.Setenv("GLOW_TEXT_SIZING", "on")
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

	rendered, err := glamourRender(m, "## Hello world heading\n")
	if err != nil {
		t.Fatalf("glamourRender: %v", err)
	}
	m.setContent(rendered)
	m.startSearch()
	m.searchInput.SetValue("Hello")
	m.confirmSearch()

	view := m.viewport.View()

	// " world" must still be wrapped in the heading's own color/bold SGR
	// code (38;5;39;1 in the dark style) immediately before its OSC 66
	// sequence, not left bare after the highlighted "Hello" resets state.
	if !strings.Contains(view, "\x1b[38;5;39;1m\x1b]66;s=2; world\x1b\\") {
		t.Fatalf("expected heading style restored before ' world', got view:\n%s", view)
	}
}

// TestSearchIgnoresLineNumberGutterDigits is the regression test for another
// reviewer-reported edge case: on code files (or markdown with
// ShowLineNumbers), each rendered line is prefixed with a styled
// line-number gutter. Since that gutter is ordinary styled text (not
// something xansi.Strip removes), searching for a short numeric string like
// "12" used to also match the gutter of line 12 (and 120-129, 212, etc.),
// not just real occurrences of "12" in the document's actual content.
func TestSearchIgnoresLineNumberGutterDigits(t *testing.T) {
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
	m.currentDocument = markdown{Note: "test.go"} // non-markdown -> gutter shown

	var md string
	for i := 1; i <= 20; i++ {
		md += fmt.Sprintf("line number %d here\n", i)
	}

	rendered, err := glamourRender(m, md)
	if err != nil {
		t.Fatalf("glamourRender: %v", err)
	}

	m.setContent(rendered)
	m.startSearch()
	m.searchInput.SetValue("12")
	m.confirmSearch()

	// Without the gutter fix this would be 2: the real "12" in "line number
	// 12 here", plus a false positive on line 12's own gutter digits.
	if len(m.searchMatches) != 1 {
		t.Fatalf("expected exactly 1 match (ignoring gutter digits), got %d: %+v", len(m.searchMatches), m.searchMatches)
	}

	view := m.viewport.View()
	wantFragment := m.common.styles.searchSelectedHighlightStyle.Render("12")
	if !strings.Contains(view, wantFragment) {
		t.Fatalf("expected the real '12' in the body text to be highlighted, got:\n%s", view)
	}
}

// TestSearchIgnoresGutterDigitsAfterStyleChanges is the regression test for
// a bug in the original gutter fix, reported by the same reviewer: it
// blanked the gutter by reconstructing and matching its exact styled text,
// which only works if m.common.styles at search time matches whatever
// Styles were current when the content was actually rendered. Those can
// genuinely differ in the running app (e.g. tea.BackgroundColorMsg replaces
// m.common.styles after the initial render), which made gutter digits leak
// through as false matches intermittently. Simulated here by changing
// styles between render and search.
func TestSearchIgnoresGutterDigitsAfterStyleChanges(t *testing.T) {
	m := newTestPagerModel()
	m.currentDocument = markdown{Note: "test.go"}

	var md string
	for i := 1; i <= 20; i++ {
		md += fmt.Sprintf("line number %d here\n", i)
	}
	rendered, err := glamourRender(m, md)
	if err != nil {
		t.Fatalf("glamourRender: %v", err)
	}
	m.setContent(rendered)

	// The content above was rendered with the dark styles newTestPagerModel
	// sets up. Now simulate the terminal's background color being detected
	// afterward, which replaces m.common.styles with a different instance
	// (light, here) before the user searches.
	m.common.styles = newStyles(false)

	m.startSearch()
	m.searchInput.SetValue("12")
	m.confirmSearch()

	if len(m.searchMatches) != 1 {
		t.Fatalf("expected exactly 1 match even after a styles change, got %d: %+v", len(m.searchMatches), m.searchMatches)
	}
}

// TestSearchFindsGutterLikeDigitsInPlainMarkdown confirms the gutter fix is
// scoped correctly: plain markdown (no line numbers shown) must still find
// numeric matches normally, since there's no gutter to confuse them with.
func TestSearchFindsGutterLikeDigitsInPlainMarkdown(t *testing.T) {
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
	m.currentDocument = markdown{Note: "test.md"} // markdown, no gutter

	rendered, err := glamourRender(m, "See line 12 for details.\n")
	if err != nil {
		t.Fatalf("glamourRender: %v", err)
	}

	m.setContent(rendered)
	m.startSearch()
	m.searchInput.SetValue("12")
	m.confirmSearch()

	if len(m.searchMatches) != 1 {
		t.Fatalf("expected 1 match in plain markdown with no gutter, got %d", len(m.searchMatches))
	}
}

// TestBlankLineNumberGutterPreservesAlignment confirms blankLineNumberGutter
// only touches the gutter prefix of each line, leaving everything else
// (including lines without a matching gutter, e.g. a mismatched/already
// mutated line) unchanged, and that the blanked gutter is the same width as
// what it replaced.
func TestBlankLineNumberGutterPreservesAlignment(t *testing.T) {
	styles := newStyles(true)
	content := styles.lineNumberStyle("   1") + "first line\n" +
		styles.lineNumberStyle("   2") + "second line"

	got := blankLineNumberGutter(content)

	want := styles.lineNumberStyle("    ") + "first line\n" +
		styles.lineNumberStyle("    ") + "second line"
	if got != want {
		t.Fatalf("blankLineNumberGutter mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

// TestBlankLineNumberGutterIgnoresStyleUsedToRenderIt is the regression test
// for a bug in the original gutter fix: it blanked the gutter by
// reconstructing and matching its exact styled text, which only worked if
// the Styles in effect when blanking ran were the same ones the content was
// originally rendered with. Those can differ in practice (e.g. if the
// terminal's background color is detected, replacing m.common.styles, in
// between a render and a search), silently leaking gutter digits through.
// Blanking by column position instead doesn't care what styled the gutter
// at all -- exercised here by using a style ("bold red") that
// lineNumberStyle would never actually produce, confirming the result
// doesn't depend on matching any particular styling.
func TestBlankLineNumberGutterIgnoresStyleUsedToRenderIt(t *testing.T) {
	unexpectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1"))
	content := unexpectedStyle.Render("   1") + "first line"

	got := blankColumnRange(content, 0, lineNumberWidth)

	want := unexpectedStyle.Render("    ") + "first line"
	if got != want {
		t.Fatalf("expected gutter blanked regardless of which style rendered it, got %q want %q", got, want)
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
