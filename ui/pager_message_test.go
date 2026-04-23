package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPagerMessageFlow(t *testing.T) {
	// Initialize config (global variable used by pager)
	config = Config{
		GlamourEnabled:        false, // Disable glamour for simpler testing
		HighPerformancePager:  false,
	}

	common := &commonModel{
		width:  80,
		height: 24,
	}

	pager := newPagerModel(common)
	pager.setSize(80, 24)

	// Simulate receiving rendered content (what happens after file load)
	testContent := `# Test Document

This document contains the word apple.

Another line with apple here.

And one more apple at the end.
`

	// Simulate contentRenderedMsg
	msg := contentRenderedMsg(testContent)
	newPager, _ := pager.update(msg)
	pager = newPager

	// Verify content was stored
	if pager.renderedContent != testContent {
		t.Error("renderedContent was not stored correctly")
		t.Logf("expected length: %d, got: %d", len(testContent), len(pager.renderedContent))
	}

	// Simulate pressing '/' to enter search mode
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	newPager, _ = pager.update(keyMsg)
	pager = newPager

	if pager.state != pagerStateSearch {
		t.Errorf("expected state pagerStateSearch, got %d", pager.state)
	}

	// Simulate typing 'apple'
	for _, r := range "apple" {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		newPager, _ = pager.update(keyMsg)
		pager = newPager
	}

	// Check that the search input has the correct value
	if pager.searchInput.Value() != "apple" {
		t.Errorf("expected searchInput value 'apple', got %q", pager.searchInput.Value())
	}

	// Simulate pressing Enter to execute search
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	newPager, _ = pager.update(enterMsg)
	pager = newPager

	// Check search was executed
	if pager.state != pagerStateBrowse {
		t.Errorf("expected state pagerStateBrowse after Enter, got %d", pager.state)
	}

	if !pager.searchActive {
		t.Error("searchActive should be true after search")
	}

	if pager.searchQuery != "apple" {
		t.Errorf("expected searchQuery 'apple', got %q", pager.searchQuery)
	}

	// Check matches were found
	if len(pager.searchMatches) != 3 {
		t.Errorf("expected 3 matches for 'apple', got %d", len(pager.searchMatches))
	}

	// Test 'n' to go to next match
	if len(pager.searchMatches) > 0 {
		pager.currentMatchIndex = 0
		nMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		newPager, _ = pager.update(nMsg)
		pager = newPager

		if pager.currentMatchIndex != 1 {
			t.Errorf("expected currentMatchIndex 1 after 'n', got %d", pager.currentMatchIndex)
		}
	}

	// Test 'N' to go to previous match
	shiftNMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}}
	newPager, _ = pager.update(shiftNMsg)
	pager = newPager

	if pager.currentMatchIndex != 0 {
		t.Errorf("expected currentMatchIndex 0 after 'N', got %d", pager.currentMatchIndex)
	}

	// Test Esc to clear search
	escMsg := tea.KeyMsg{Type: tea.KeyEscape}
	newPager, _ = pager.update(escMsg)
	pager = newPager

	if pager.searchActive {
		t.Error("searchActive should be false after Esc")
	}

	if len(pager.searchMatches) != 0 {
		t.Error("searchMatches should be empty after Esc")
	}
}

func TestSearchInputKeyHandling(t *testing.T) {
	config = Config{
		GlamourEnabled:       false,
		HighPerformancePager: false,
	}

	common := &commonModel{
		width:  80,
		height: 24,
	}

	pager := newPagerModel(common)
	pager.setSize(80, 24)
	pager.setRenderedContent("test content with apple")
	pager.viewport.SetContent(pager.renderedContent)

	// Enter search mode
	pager.state = pagerStateSearch
	pager.searchInput.Focus()

	// Type some characters
	for _, r := range "test" {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		newPager, _ := pager.update(keyMsg)
		pager = newPager
	}

	if pager.searchInput.Value() != "test" {
		t.Errorf("expected 'test', got %q", pager.searchInput.Value())
	}

	// Press Esc to cancel
	escMsg := tea.KeyMsg{Type: tea.KeyEscape}
	newPager, _ := pager.update(escMsg)
	pager = newPager

	if pager.state != pagerStateBrowse {
		t.Errorf("expected pagerStateBrowse after Esc, got %d", pager.state)
	}

	// The search input should be reset
	if pager.searchInput.Value() != "" {
		t.Errorf("searchInput should be empty after Esc cancel, got %q", pager.searchInput.Value())
	}
}
