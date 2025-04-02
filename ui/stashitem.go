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
		truncateTo  = uint(m.common.width - stashViewHorizontalPadding*2) //nolint:gosec
		gutter      string
		title       = truncate.StringWithTail(md.Note, truncateTo, ellipsis)
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
	if isSelected && !isFiltering || singleFilteredItem { //nolint:nestif
		// Selected item
		if m.statusMessage == stashingStatusMessage {
			gutter = greenFg(verticalLine)
			icon = dimGreenFg(icon)
			title = greenFg(title)
			date = semiDimGreenFg(date)
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
			editedBy = dimDullFuchsiaFg(editedBy)
			separator = dullFuchsiaFg(separator)
		}
	} else {
		gutter = " "
		if m.statusMessage == stashingStatusMessage {
			icon = dimGreenFg(icon)
			title = greenFg(title)
			date = semiDimGreenFg(date)
			editedBy = semiDimGreenFg(editedBy)
			separator = semiDimGreenFg(separator)
		} else if isFiltering && m.filterInput.Value() == "" {
			icon = dimGreenFg(icon)
			title = dimNormalFg(title)
			date = dimBrightGrayFg(date)
			editedBy = dimBrightGrayFg(editedBy)
			separator = dimBrightGrayFg(separator)
		} else {
			icon = greenFg(icon)

			s := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"})
			title = styleFilteredText(title, m.filterInput.Value(), s, s.Underline(true))
			date = grayFg(date)
			editedBy = midGrayFg(editedBy)
			separator = brightGrayFg(separator)
		}
	}

	fmt.Fprintf(b, "%s %s%s%s%s\n", gutter, icon, separator, separator, title)
	fmt.Fprintf(b, "%s %s", gutter, date)
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
