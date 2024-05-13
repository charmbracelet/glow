package ui

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/ansi"
	"github.com/muesli/reflow/truncate"
	"github.com/sahilm/fuzzy"
)

const (
	verticalLine         = "│"
	noMemoTitle          = "No Memo"
	fileListingStashIcon = "• "
)

func stashItemView(b *strings.Builder, m stashModel, index int, md *markdown) {
	var (
		truncateTo = uint(m.common.width - stashViewHorizontalPadding*2)
		gutter     string
		title      = md.Note
		date       = md.relativeTime()
		icon       = ""
	)

	switch md.docType {
	case NewsDoc:
		if title == "" {
			title = "News"
		} else {
			title = truncate.StringWithTail(title, truncateTo, ellipsis)
		}
	case StashedDoc, ConvertedDoc:
		icon = fileListingStashIcon
		if title == "" {
			title = noMemoTitle
		}
		title = truncate.StringWithTail(title, truncateTo-uint(ansi.PrintableRuneWidth(icon)), ellipsis)
	default:
		title = truncate.StringWithTail(title, truncateTo, ellipsis)
	}

	isSelected := index == m.cursor()
	isFiltering := m.filterState == filtering
	singleFilteredItem := isFiltering && len(m.getVisibleMarkdowns()) == 1
	styles := &m.common.styles

	// If there are multiple items being filtered don't highlight a selected
	// item in the results. If we've filtered down to one item, however,
	// highlight that first item since pressing return will open it.
	if isSelected && !isFiltering || singleFilteredItem {
		// Selected item

		switch m.selectionState {
		case selectionPromptingDelete:
			gutter = styles.FaintRedFg(verticalLine)
			icon = styles.FaintRedFg(icon)
			title = styles.RedFg(title)
			date = styles.FaintRedFg(date)
		case selectionSettingNote:
			gutter = styles.DullYellowFg(verticalLine)
			icon = ""
			title = m.noteInput.View(m.common.ctx)
			date = styles.DullYellowFg(date)
		default:
			if m.common.latestFileStashed == md.stashID &&
				m.statusMessage == stashingStatusMessage {
				gutter = styles.GreenFg(verticalLine)
				icon = styles.DimGreenFg(icon)
				title = styles.GreenFg(title)
				date = styles.SemiDimGreenFg(date)
			} else {
				gutter = styles.DullFuchsiaFg(verticalLine)
				icon = styles.DullFuchsiaFg(icon)
				if m.currentSection().key == filterSection &&
					m.filterState == filterApplied || singleFilteredItem {
					s := lipgloss.NewStyle().Foreground(fuchsia)
					title = styleFilteredText(title, m.filterInput.Value(), s, s.Copy().Underline(true))
				} else {
					title = styles.FuchsiaFg(title)
				}
				date = styles.DullFuchsiaFg(date)
			}
		}
	} else {
		// Regular (non-selected) items

		gutter = " "

		if m.common.latestFileStashed == md.stashID &&
			m.statusMessage == stashingStatusMessage {
			icon = styles.DimGreenFg(icon)
			title = styles.GreenFg(title)
			date = styles.SemiDimGreenFg(date)
		} else if md.docType == NewsDoc {
			if isFiltering && m.filterInput.Value() == "" {
				title = styles.DimIndigoFg(title)
				date = styles.DimSubtleIndigoFg(date)
			} else {
				s := lipgloss.NewStyle().Foreground(indigo)
				title = styleFilteredText(title, m.filterInput.Value(), s, s.Copy().Underline(true))
				date = styles.SubtleIndigoFg(date)
			}
		} else if isFiltering && m.filterInput.Value() == "" {
			icon = styles.DimGreenFg(icon)
			if title == noMemoTitle {
				title = styles.DimBrightGrayFg(title)
			} else {
				title = styles.DimNormalFg(title)
			}
			date = styles.DimBrightGrayFg(date)
		} else {
			icon = styles.GreenFg(icon)
			if title == noMemoTitle {
				title = styles.BrightGrayFg(title)
			} else {
				s := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"})
				title = styleFilteredText(title, m.filterInput.Value(), s, s.Copy().Underline(true))
			}
			date = styles.BrightGrayFg(date)
		}
	}

	fmt.Fprintf(b, "%s %s%s\n", gutter, icon, title)
	fmt.Fprintf(b, "%s %s", gutter, date)
}

func styleFilteredText(haystack, needles string, defaultStyle, matchedStyle lipgloss.Style) string {
	b := strings.Builder{}

	normalizedHay, err := normalize(haystack)
	if err != nil && debug {
		log.Printf("error normalizing '%s': %v", haystack, err)
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
