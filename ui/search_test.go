package ui

import (
	"regexp"
	"strings"
	"testing"
)

func TestStripAnsi(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"\x1b[31mred\x1b[0m", "red"},
		{"\x1b[1;32mbold green\x1b[0m text", "bold green text"},
		{"no ansi here", "no ansi here"},
	}

	for _, tt := range tests {
		result := stripAnsi(tt.input)
		if result != tt.expected {
			t.Errorf("stripAnsi(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSearchInContent(t *testing.T) {
	content := `# Test Document

This is a test with apple in it.

Another line with APPLE (uppercase).

And one more apple here.`

	lines := strings.Split(content, "\n")
	query := "apple"
	escapedQuery := regexp.QuoteMeta(query)
	re, err := regexp.Compile("(?i)" + escapedQuery)
	if err != nil {
		t.Fatalf("failed to compile regex: %v", err)
	}

	var matches []searchMatch
	for lineIdx, line := range lines {
		plainLine := stripAnsi(line)
		found := re.FindAllStringIndex(plainLine, -1)
		for _, match := range found {
			matches = append(matches, searchMatch{
				lineIndex: lineIdx,
				startCol:  match[0],
				endCol:    match[1],
			})
		}
	}

	// Should find 3 matches for "apple"
	if len(matches) != 3 {
		t.Errorf("expected 3 matches for 'apple', got %d", len(matches))
		for i, m := range matches {
			t.Logf("match %d: line %d, col %d-%d", i, m.lineIndex, m.startCol, m.endCol)
		}
	}

	// Verify line numbers
	expectedLines := []int{2, 4, 6}
	for i, m := range matches {
		if m.lineIndex != expectedLines[i] {
			t.Errorf("match %d: expected line %d, got %d", i, expectedLines[i], m.lineIndex)
		}
	}
}

func TestSearchWithAnsiContent(t *testing.T) {
	// Simulate glamour-rendered content with ANSI codes
	content := "\x1b[1m# Test\x1b[0m\n\nThis has \x1b[32mapple\x1b[0m in it.\n\nAnother \x1b[33mAPPLE\x1b[0m here."

	lines := strings.Split(content, "\n")
	query := "apple"
	escapedQuery := regexp.QuoteMeta(query)
	re, err := regexp.Compile("(?i)" + escapedQuery)
	if err != nil {
		t.Fatalf("failed to compile regex: %v", err)
	}

	var matches []searchMatch
	for lineIdx, line := range lines {
		plainLine := stripAnsi(line)
		found := re.FindAllStringIndex(plainLine, -1)
		for _, match := range found {
			matches = append(matches, searchMatch{
				lineIndex: lineIdx,
				startCol:  match[0],
				endCol:    match[1],
			})
		}
	}

	// Should find 2 matches for "apple" even with ANSI codes
	if len(matches) != 2 {
		t.Errorf("expected 2 matches for 'apple', got %d", len(matches))
		for i, m := range matches {
			plainLine := stripAnsi(lines[m.lineIndex])
			t.Logf("match %d: line %d (%q), col %d-%d", i, m.lineIndex, plainLine, m.startCol, m.endCol)
		}
	}
}
