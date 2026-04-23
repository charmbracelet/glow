package ui

import (
	"strings"
	"testing"
)

func TestPagerSearchIntegration(t *testing.T) {
	// Create a mock common model
	common := &commonModel{
		width:  80,
		height: 24,
	}

	// Create a new pager model
	pager := newPagerModel(common)
	pager.setSize(80, 24)

	// Simulate rendered content being set (like what happens after glamour rendering)
	renderedContent := `# Search Test Document

This is a test document for testing the search functionality.

## Section One

The quick brown fox jumps over the lazy dog.

Here is the word apple in a sentence.

## Section Two

Another paragraph with apple mentioned again.

- Item with apple
- Item with banana

## Section Three

Final apple reference here.
`

	// Set the rendered content (this normally happens via contentRenderedMsg)
	pager.setRenderedContent(renderedContent)
	pager.viewport.SetContent(renderedContent)

	// Test search for "apple"
	pager.searchQuery = "apple"
	pager.performSearch()

	// Verify matches were found
	if len(pager.searchMatches) != 4 {
		t.Errorf("expected 4 matches for 'apple', got %d", len(pager.searchMatches))
		for i, m := range pager.searchMatches {
			t.Logf("match %d: line %d", i, m.lineIndex)
		}
	}

	if !pager.searchActive {
		t.Error("searchActive should be true after search")
	}

	// Test jumping to match
	if len(pager.searchMatches) > 0 {
		pager.currentMatchIndex = 0
		pager.jumpToCurrentMatch()

		// The viewport should have moved
		match := pager.searchMatches[0]
		t.Logf("first match at line %d, viewport offset: %d", match.lineIndex, pager.viewport.YOffset)
	}

	// Test clear search
	pager.clearSearch()
	if pager.searchActive {
		t.Error("searchActive should be false after clearSearch")
	}
	if len(pager.searchMatches) != 0 {
		t.Error("searchMatches should be empty after clearSearch")
	}
	if pager.currentMatchIndex != -1 {
		t.Error("currentMatchIndex should be -1 after clearSearch")
	}
}

func TestPagerSearchNoMatches(t *testing.T) {
	common := &commonModel{
		width:  80,
		height: 24,
	}

	pager := newPagerModel(common)
	pager.setSize(80, 24)

	pager.setRenderedContent("This is some content without the search term.")
	pager.viewport.SetContent(pager.renderedContent)

	pager.searchQuery = "nonexistent"
	pager.performSearch()

	if len(pager.searchMatches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(pager.searchMatches))
	}

	if !pager.searchActive {
		t.Error("searchActive should be true even with no matches")
	}
}

func TestPagerSearchCaseInsensitive(t *testing.T) {
	common := &commonModel{
		width:  80,
		height: 24,
	}

	pager := newPagerModel(common)
	pager.setSize(80, 24)

	pager.setRenderedContent("Apple APPLE apple ApPlE")
	pager.viewport.SetContent(pager.renderedContent)

	pager.searchQuery = "apple"
	pager.performSearch()

	if len(pager.searchMatches) != 4 {
		t.Errorf("expected 4 case-insensitive matches, got %d", len(pager.searchMatches))
	}
}

func TestPagerSearchSmartCaseSensitive(t *testing.T) {
	common := &commonModel{
		width:  80,
		height: 24,
	}

	pager := newPagerModel(common)
	pager.setSize(80, 24)

	// Content with multiple case variations
	pager.setRenderedContent("Apple APPLE apple ApPlE")
	pager.viewport.SetContent(pager.renderedContent)

	// Search with uppercase "Apple" - smart case should make this case-sensitive
	pager.searchQuery = "Apple"
	pager.performSearch()

	// Should only match "Apple" (exact case), not "APPLE", "apple", or "ApPlE"
	if len(pager.searchMatches) != 1 {
		t.Errorf("expected 1 case-sensitive match for 'Apple', got %d", len(pager.searchMatches))
	}

	// Verify the match is at the correct position (start of string)
	if len(pager.searchMatches) > 0 {
		match := pager.searchMatches[0]
		if match.startCol != 0 || match.endCol != 5 {
			t.Errorf("expected match at columns 0-5, got %d-%d", match.startCol, match.endCol)
		}
	}
}

func TestPagerSearchHighlightingMatchesSearch(t *testing.T) {
	common := &commonModel{
		width:  80,
		height: 24,
	}

	pager := newPagerModel(common)
	pager.setSize(80, 24)

	// Multi-line content with case variations
	pager.setRenderedContent("Line1: Apple here\nLine2: APPLE here\nLine3: apple here\nLine4: ApPlE here")
	pager.viewport.SetContent(pager.renderedContent)

	// Case-sensitive search for "Apple"
	pager.searchQuery = "Apple"
	pager.performSearch()

	// performSearch should find 1 match
	if len(pager.searchMatches) != 1 {
		t.Errorf("performSearch: expected 1 match, got %d", len(pager.searchMatches))
	}

	// Verify match is on line 0 (Line1: Apple here)
	if len(pager.searchMatches) > 0 && pager.searchMatches[0].lineIndex != 0 {
		t.Errorf("expected match on line 0, got line %d", pager.searchMatches[0].lineIndex)
	}
}

func TestPagerSearchWithAnsiCodes(t *testing.T) {
	common := &commonModel{
		width:  80,
		height: 24,
	}

	pager := newPagerModel(common)
	pager.setSize(80, 24)

	// Simulate content with ANSI color codes (like glamour output)
	pager.setRenderedContent("\x1b[1m# Header\x1b[0m\n\nThis has \x1b[32mapple\x1b[0m in it.\n\nAnother \x1b[33mAPPLE\x1b[0m here.")
	pager.viewport.SetContent(pager.renderedContent)

	pager.searchQuery = "apple"
	pager.performSearch()

	if len(pager.searchMatches) != 2 {
		t.Errorf("expected 2 matches with ANSI codes, got %d", len(pager.searchMatches))
	}
}

func TestPagerNextPrevMatch(t *testing.T) {
	common := &commonModel{
		width:  80,
		height: 24,
	}

	pager := newPagerModel(common)
	pager.setSize(80, 24)

	pager.setRenderedContent("apple\napple\napple")
	pager.viewport.SetContent(pager.renderedContent)

	pager.searchQuery = "apple"
	pager.performSearch()
	pager.currentMatchIndex = 0

	// Test next
	pager.currentMatchIndex = (pager.currentMatchIndex + 1) % len(pager.searchMatches)
	if pager.currentMatchIndex != 1 {
		t.Errorf("expected currentMatchIndex 1, got %d", pager.currentMatchIndex)
	}

	// Test wrap around
	pager.currentMatchIndex = (pager.currentMatchIndex + 1) % len(pager.searchMatches)
	pager.currentMatchIndex = (pager.currentMatchIndex + 1) % len(pager.searchMatches)
	if pager.currentMatchIndex != 0 {
		t.Errorf("expected currentMatchIndex to wrap to 0, got %d", pager.currentMatchIndex)
	}

	// Test prev
	pager.currentMatchIndex--
	if pager.currentMatchIndex < 0 {
		pager.currentMatchIndex = len(pager.searchMatches) - 1
	}
	if pager.currentMatchIndex != 2 {
		t.Errorf("expected currentMatchIndex 2 after prev from 0, got %d", pager.currentMatchIndex)
	}
}

func TestHighlightRenderedLinePreservesFormatting(t *testing.T) {
	// Test that ANSI formatting is preserved in non-matched portions of the line
	// This is the core fix for the bug where updateSearchHighlighting was
	// reconstructing lines from plain text and losing all ANSI formatting

	// Line with ANSI formatting: "prefix " (green) + "word" (normal) + " suffix" (blue)
	renderedLine := "\x1b[32mprefix \x1b[0mword\x1b[34m suffix\x1b[0m"

	// Match "word" which is at plain-text positions 7-11
	matches := []searchMatch{{lineIndex: 0, startCol: 7, endCol: 11}}

	result := highlightRenderedLine(renderedLine, matches, -1, 0)

	// The result should:
	// 1. Preserve the green ANSI code before "word"
	// 2. Have the highlight applied to "word"
	// 3. Preserve the blue ANSI code after "word"

	// Check that green code is preserved at the start
	if !strings.Contains(result, "\x1b[32m") {
		t.Error("expected green ANSI code to be preserved in the result")
	}

	// Check that blue code is preserved at the end
	if !strings.Contains(result, "\x1b[34m") {
		t.Error("expected blue ANSI code to be preserved in the result")
	}

	// Check that "prefix " appears in the result
	plainResult := stripAnsi(result)
	if !strings.Contains(plainResult, "prefix ") {
		t.Errorf("expected 'prefix ' in result, got: %s", plainResult)
	}

	// Check that "word" appears in the result
	if !strings.Contains(plainResult, "word") {
		t.Errorf("expected 'word' in result, got: %s", plainResult)
	}

	// Check that " suffix" appears in the result
	if !strings.Contains(plainResult, " suffix") {
		t.Errorf("expected ' suffix' in result, got: %s", plainResult)
	}
}
