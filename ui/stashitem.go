package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
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
	styles := m.common.styles

	// If there are multiple items being filtered don't highlight a selected
	// item in the results. If we've filtered down to one item, however,
	// highlight that first item since pressing return will open it.
	if isSelected && !isFiltering || singleFilteredItem { //nolint:nestif
		// Selected item
		if m.statusMessage == stashingStatusMessage {
			gutter = styles.greenFg(verticalLine)
			icon = styles.dimGreenFg(icon)
			title = styles.greenFg(title)
			date = styles.semiDimGreenFg(date)
			editedBy = styles.semiDimGreenFg(editedBy)
			separator = styles.semiDimGreenFg(separator)
		} else {
			gutter = styles.dullFuchsiaFg(verticalLine)
			if m.currentSection().key == filterSection &&
				m.filterState == filterApplied || singleFilteredItem {
				s := lipgloss.NewStyle().Foreground(styles.fuchsia)
				title = styleFilteredText(title, m.filterInput.Value(), s, s.Underline(true))
			} else {
				title = styles.fuchsiaFg(title)
				icon = styles.fuchsiaFg(icon)
			}
			date = styles.dimFuchsiaFg(date)
			editedBy = styles.dimDullFuchsiaFg(editedBy)
			separator = styles.dullFuchsiaFg(separator)
		}
	} else {
		gutter = " "
		if m.statusMessage == stashingStatusMessage {
			icon = styles.dimGreenFg(icon)
			title = styles.greenFg(title)
			date = styles.semiDimGreenFg(date)
			editedBy = styles.semiDimGreenFg(editedBy)
			separator = styles.semiDimGreenFg(separator)
		} else if isFiltering && m.filterInput.Value() == "" {
			icon = styles.dimGreenFg(icon)
			title = styles.dimNormalFg(title)
			date = styles.dimBrightGrayFg(date)
			editedBy = styles.dimBrightGrayFg(editedBy)
			separator = styles.dimBrightGrayFg(separator)
		} else {
			icon = styles.greenFg(icon)

			s := lipgloss.NewStyle().Foreground(styles.adaptive("#1a1a1a", "#dddddd"))
			title = styleFilteredText(title, m.filterInput.Value(), s, s.Underline(true))
			date = styles.grayFg(date)
			editedBy = styles.midGrayFg(editedBy)
			separator = styles.brightGrayFg(separator)
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
