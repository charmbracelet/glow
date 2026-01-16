package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestIncrementalSearch(t *testing.T) {
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

	testContent := `Line 1: nothing here
Line 2: apple is here
Line 3: more content
Line 4: another apple
Line 5: the end
`

	pager.setRenderedContent(testContent)
	pager.viewport.SetContent(testContent)

	// Enter search mode
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	newPager, _ := pager.update(keyMsg)
	pager = newPager

	if pager.state != pagerStateSearch {
		t.Fatalf("expected pagerStateSearch, got %d", pager.state)
	}

	// Type 'a' - should find matches incrementally
	aMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	newPager, _ = pager.update(aMsg)
	pager = newPager

	// After typing 'a', we should have matches (apple has 'a')
	if pager.searchInput.Value() != "a" {
		t.Errorf("expected searchInput 'a', got %q", pager.searchInput.Value())
	}
	if !pager.searchActive {
		t.Error("searchActive should be true after typing")
	}
	// 'a' appears in 'apple' (2 times in lines 2 and 4), plus 'another' and 'the end' 'a'
	if len(pager.searchMatches) < 1 {
		t.Errorf("expected at least 1 match for 'a', got %d", len(pager.searchMatches))
	}

	// Continue typing 'pple' to form 'apple'
	for _, r := range "pple" {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		newPager, _ = pager.update(msg)
		pager = newPager
	}

	if pager.searchInput.Value() != "apple" {
		t.Errorf("expected searchInput 'apple', got %q", pager.searchInput.Value())
	}

	// Should have exactly 2 matches for 'apple'
	if len(pager.searchMatches) != 2 {
		t.Errorf("expected 2 matches for 'apple', got %d", len(pager.searchMatches))
	}

	// Current match should be 0 (first match)
	if pager.currentMatchIndex != 0 {
		t.Errorf("expected currentMatchIndex 0, got %d", pager.currentMatchIndex)
	}

	// Press Enter to confirm search
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	newPager, _ = pager.update(enterMsg)
	pager = newPager

	if pager.state != pagerStateBrowse {
		t.Errorf("expected pagerStateBrowse after Enter, got %d", pager.state)
	}

	// Search should still be active
	if !pager.searchActive {
		t.Error("searchActive should still be true after confirming")
	}

	// Matches should still be there
	if len(pager.searchMatches) != 2 {
		t.Errorf("expected 2 matches after confirm, got %d", len(pager.searchMatches))
	}
}

func TestIncrementalSearchEscCancels(t *testing.T) {
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

	testContent := "content with apple here"
	pager.setRenderedContent(testContent)
	pager.viewport.SetContent(testContent)

	// Enter search mode
	slashMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	newPager, _ := pager.update(slashMsg)
	pager = newPager

	// Type 'app'
	for _, r := range "app" {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		newPager, _ = pager.update(msg)
		pager = newPager
	}

	// Should have a match
	if len(pager.searchMatches) == 0 {
		t.Error("expected matches for 'app'")
	}

	// Press Esc to cancel
	escMsg := tea.KeyMsg{Type: tea.KeyEscape}
	newPager, _ = pager.update(escMsg)
	pager = newPager

	// Should be back to browse state
	if pager.state != pagerStateBrowse {
		t.Errorf("expected pagerStateBrowse after Esc, got %d", pager.state)
	}

	// Search should be cleared
	if pager.searchActive {
		t.Error("searchActive should be false after Esc")
	}
	if len(pager.searchMatches) != 0 {
		t.Error("searchMatches should be empty after Esc")
	}
	if pager.searchInput.Value() != "" {
		t.Errorf("searchInput should be empty after Esc, got %q", pager.searchInput.Value())
	}
}

func TestIncrementalSearchEmptyQuery(t *testing.T) {
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

	testContent := "test content"
	pager.setRenderedContent(testContent)
	pager.viewport.SetContent(testContent)

	// Enter search mode
	slashMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	newPager, _ := pager.update(slashMsg)
	pager = newPager

	// Type something
	for _, r := range "test" {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		newPager, _ = pager.update(msg)
		pager = newPager
	}

	if !pager.searchActive {
		t.Error("searchActive should be true")
	}

	// Delete all characters using backspace
	for i := 0; i < 4; i++ {
		bsMsg := tea.KeyMsg{Type: tea.KeyBackspace}
		newPager, _ = pager.update(bsMsg)
		pager = newPager
	}

	// With empty query, search should be inactive
	if pager.searchActive {
		t.Error("searchActive should be false with empty query")
	}
	if len(pager.searchMatches) != 0 {
		t.Error("searchMatches should be empty with empty query")
	}
}
