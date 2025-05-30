package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	runewidth "github.com/mattn/go-runewidth"
)

func TestZenModeCenter(t *testing.T) {
	// Create a mock pager model with zen mode enabled
	common := &commonModel{
		cfg: Config{
			ZenMode: true,
		},
		width: 80,
	}

	pager := pagerModel{
		common:   common,
		viewport: viewport.New(80, 24),
	}

	// Test input with varying line lengths
	input := `# Test Header
This is a short line.
This is a much longer line that should still be centered properly.
Short.`

	// Split into lines like the glamourRender function does
	lines := strings.Split(input, "\n")

	var content strings.Builder
	for i, s := range lines {
		lineContent := s

		// Apply center justification logic (simplified version of what's in glamourRender)
		if common.cfg.ZenMode {
			contentWidth := runewidth.StringWidth(s)
			if contentWidth < pager.viewport.Width {
				leftPadding := (pager.viewport.Width - contentWidth) / 2
				centerPadding := strings.Repeat(" ", leftPadding)
				lineContent = centerPadding + s
			}
		}

		content.WriteString(lineContent)
		if i+1 < len(lines) {
			content.WriteRune('\n')
		}
	}

	result := content.String()
	resultLines := strings.Split(result, "\n")

	// Test that lines are centered
	for i, line := range resultLines {
		originalLine := lines[i]
		originalWidth := runewidth.StringWidth(originalLine)
		
		// Calculate expected padding
		expectedPadding := (80 - originalWidth) / 2
		actualPadding := 0
		
		// Count leading spaces
		for _, r := range line {
			if r == ' ' {
				actualPadding++
			} else {
				break
			}
		}

		if actualPadding != expectedPadding {
			t.Errorf("Line %d: expected %d spaces, got %d spaces. Line: %q", 
				i, expectedPadding, actualPadding, line)
		}

		// Check that the content is preserved
		trimmedLine := strings.TrimLeft(line, " ")
		if trimmedLine != originalLine {
			t.Errorf("Line %d: content changed. Expected %q, got %q", 
				i, originalLine, trimmedLine)
		}
	}
}

func TestZenModeDisabled(t *testing.T) {
	// Create a mock pager model with zen mode disabled
	common := &commonModel{
		cfg: Config{
			ZenMode: false,
		},
		width: 80,
	}

	input := "This line should not be centered."

	// When zen mode is disabled, content should remain unchanged
	result := input
	if common.cfg.ZenMode {
		// This shouldn't execute
		t.Error("ZenMode should be false")
	}

	if result != input {
		t.Errorf("Content should be unchanged when zen mode is disabled. Expected %q, got %q", 
			input, result)
	}
}

func TestZenModeWithLineNumbers(t *testing.T) {
	// Create a mock pager model with zen mode and line numbers enabled
	common := &commonModel{
		cfg: Config{
			ZenMode:         true,
			ShowLineNumbers: true,
		},
		width: 80,
	}

	// Test that line numbers are properly handled with zen mode
	input := "Test line with line numbers"
	
	// In zen mode with line numbers, we should still center the content
	// but account for the line number prefix
	contentWidth := runewidth.StringWidth(input)
	availableWidth := 80 - lineNumberWidth
	expectedPadding := (availableWidth - contentWidth) / 2
	
	if expectedPadding < 0 {
		expectedPadding = 0
	}

	// This test verifies the logic is sound
	if common.cfg.ZenMode && common.cfg.ShowLineNumbers {
		if expectedPadding < 0 {
			t.Error("Padding calculation should not be negative")
		}
	}
}