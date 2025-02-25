package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/muesli/reflow/truncate"
	"github.com/sahilm/fuzzy"
)

const (
	verticalLine         = "│"
	fileListingStashIcon = "• "
)

func stashItemView(b *strings.Builder, m stashModel, index int, md *markdown) {
	var (
		truncateTo  = uint(m.common.width - stashViewHorizontalPadding*2)
		gutter      string
		title       = truncate.StringWithTail(md.Note, truncateTo, ellipsis)
		matchString = "Filter Matches"
		date        = md.relativeTime()
		editedBy    = ""
		hasEditedBy = false
		icon        = ""
		separator   = ""
	)

	isSelected := index == m.cursor()
	isFiltering := m.filterState == filtering
	singleFilteredItem := isFiltering && len(m.getVisibleMarkdowns()) == 1

	// If there are multiple items being filtered don't highlight a selected
	// item in the results. If we've filtered down to one item, however,
	// highlight that first item since pressing return will open it.
	if isSelected && !isFiltering || singleFilteredItem {
		// Selected item
		if m.statusMessage == stashingStatusMessage {
			gutter = greenFg(verticalLine)
			icon = dimGreenFg(icon)
			title = greenFg(title)
			date = semiDimGreenFg(date)
			matchString = semiDimGreenFg(matchString)
			editedBy = semiDimGreenFg(editedBy)
			separator = semiDimGreenFg(separator)
		} else {
			gutter = dullFuchsiaFg(verticalLine)
			if m.currentSection().key == filterSection &&
				m.filterState == filterApplied || singleFilteredItem {
				s := lipgloss.NewStyle().Foreground(fuchsia)
				title = styleFilteredText(title, m.filterInput.Value(), s, s.Underline(true))
			} else {
				title = fuchsiaFg(title)
				icon = fuchsiaFg(icon)
			}
			date = dimFuchsiaFg(date)
			matchString = dimFuchsiaFg(matchString)
			editedBy = dimDullFuchsiaFg(editedBy)
			separator = dullFuchsiaFg(separator)
		}
	} else {
		gutter = " "
		if m.statusMessage == stashingStatusMessage {
			icon = dimGreenFg(icon)
			title = greenFg(title)
			date = semiDimGreenFg(date)
			matchString = semiDimGreenFg(matchString)
			editedBy = semiDimGreenFg(editedBy)
			separator = semiDimGreenFg(separator)
		} else if isFiltering && m.filterInput.Value() == "" {
			icon = dimGreenFg(icon)
			title = dimNormalFg(title)
			date = dimBrightGrayFg(date)
			matchString = dimBrightGrayFg(matchString)
			editedBy = dimBrightGrayFg(editedBy)
			separator = dimBrightGrayFg(separator)
		} else {
			icon = greenFg(icon)

			s := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"})
			title = styleFilteredText(title, m.filterInput.Value(), s, s.Underline(true))
			date = grayFg(date)
			matchString = greenFg(matchString)
			editedBy = midGrayFg(editedBy)
			separator = brightGrayFg(separator)
		}
	}

	fmt.Fprintf(b, "%s %s%s%s%s\n", gutter, icon, separator, separator, title)
	fmt.Fprintf(b, "%s %s", gutter, date)
	if len(md.Matches) > 0 {
		s := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#ff5050", Dark: "#ff5050"})
		fmt.Fprintf(b, "\n%s %s (Showing %d out of %d): ", gutter, matchString, len(md.Matches), md.TotalMatchesCount)
		for _, match := range md.Matches {
			var termIndex int = strings.Index(strings.ToLower(match), strings.ToLower(m.filterInput.Value()))
			if termIndex != -1 {
				availableWidth := m.common.width - stashViewHorizontalPadding*2
				if len(match) > availableWidth {
					match = fmt.Sprintf("%s %s", "...", match[termIndex:len(match)])
					termIndex = strings.Index(strings.ToLower(match), strings.ToLower(m.filterInput.Value()))
				}
				fmt.Fprintf(b, "\n%s   ", gutter)
				for i := 0; i < len(match); i++ {
					if i >= termIndex && i < termIndex+len(m.filterInput.Value()) {
						b.WriteString(s.Render(string(match[i])))
					} else {
						fmt.Fprintf(b, "%s", string(match[i]))
					}
				}
			}
		}
	}
	if hasEditedBy {
		fmt.Fprintf(b, " %s", editedBy)
	}
}

func styleFilteredText(haystack, needles string, defaultStyle, matchedStyle lipgloss.Style) string {
	b := strings.Builder{}

	normalizedHay, err := normalize(haystack)
	if err != nil {
		log.Error("error normalizing", "haystack", haystack, "error", err)
	}

	matches := fuzzy.Find(needles, []string{normalizedHay})
	if len(matches) == 0 {
		return defaultStyle.Render(haystack)
	}

	m := matches[0] // only one match exists
	for i, rune := range []rune(haystack) {
		styled := false
		for _, mi := range m.MatchedIndexes {
			if i == mi {
				b.WriteString(matchedStyle.Render(string(rune)))
				styled = true
			}
		}
		if !styled {
			b.WriteString(defaultStyle.Render(string(rune)))
		}
	}

	return b.String()
}
