package ui

import (
	"fmt"
	"log"
	"strings"

	lib "github.com/charmbracelet/charm/ui/common"
	"github.com/muesli/reflow/ansi"
	"github.com/muesli/reflow/truncate"
	"github.com/muesli/termenv"
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

	// If there are multiple items being filtered don't highlight a selected
	// item in the results. If we've filtered down to one item, however,
	// highlight that first item since pressing return will open it.
	if isSelected && !isFiltering || singleFilteredItem {
		// Selected item

		switch m.selectionState {
		case selectionPromptingDelete:
			gutter = faintRedFg(verticalLine)
			icon = faintRedFg(icon)
			title = redFg(title)
			date = faintRedFg(date)
		case selectionSettingNote:
			gutter = dullYellowFg(verticalLine)
			icon = ""
			title = m.noteInput.View()
			date = dullYellowFg(date)
		default:
			if m.common.latestFileStashed == md.stashID &&
				m.statusMessage == stashedStatusMessage {
				gutter = greenFg(verticalLine)
				icon = dimGreenFg(icon)
				title = greenFg(title)
				date = semiDimGreenFg(date)
			} else {
				gutter = dullFuchsiaFg(verticalLine)
				icon = dullFuchsiaFg(icon)
				if m.currentSection().key == filterSection &&
					m.filterState == filterApplied || singleFilteredItem {
					s := termenv.Style{}.Foreground(lib.Fuschia.Color())
					title = styleFilteredText(title, m.filterInput.Value(), s, s.Underline())
				} else {
					title = fuchsiaFg(title)
				}
				date = dullFuchsiaFg(date)
			}
		}
	} else {
		// Regular (non-selected) items

		gutter = " "

		if m.common.latestFileStashed == md.stashID &&
			m.statusMessage == stashedStatusMessage {
			icon = dimGreenFg(icon)
			title = greenFg(title)
			date = semiDimGreenFg(date)
		} else if md.docType == NewsDoc {
			if isFiltering && m.filterInput.Value() == "" {
				title = dimIndigoFg(title)
				date = dimSubtleIndigoFg(date)
			} else {
				s := termenv.Style{}.Foreground(lib.Indigo.Color())
				title = styleFilteredText(title, m.filterInput.Value(), s, s.Underline())
				date = subtleIndigoFg(date)
			}
		} else if isFiltering && m.filterInput.Value() == "" {
			icon = dimGreenFg(icon)
			if title == noMemoTitle {
				title = dimBrightGrayFg(title)
			} else {
				title = dimNormalFg(title)
			}
			date = dimBrightGrayFg(date)
		} else {
			icon = greenFg(icon)
			if title == noMemoTitle {
				title = brightGrayFg(title)
			} else {
				s := termenv.Style{}.Foreground(lib.NewColorPair("#dddddd", "#1a1a1a").Color())
				title = styleFilteredText(title, m.filterInput.Value(), s, s.Underline())
			}
			date = brightGrayFg(date)
		}
	}

	fmt.Fprintf(b, "%s %s%s\n", gutter, icon, title)
	fmt.Fprintf(b, "%s %s", gutter, date)
}

func styleFilteredText(haystack, needles string, defaultStyle, matchedStyle termenv.Style) string {
	b := strings.Builder{}

	normalizedHay, err := normalize(haystack)
	if err != nil && debug {
		log.Printf("error normalizing '%s': %v", haystack, err)
	}

	matches := fuzzy.Find(needles, []string{normalizedHay})
	if len(matches) == 0 {
		return defaultStyle.Styled(haystack)
	}

	m := matches[0] // only one match exists
	for i, rune := range []rune(haystack) {
		styled := false
		for _, mi := range m.MatchedIndexes {
			if i == mi {
				b.WriteString(matchedStyle.Styled(string(rune)))
				styled = true
			}
		}
		if !styled {
			b.WriteString(defaultStyle.Styled(string(rune)))
		}
	}

	return b.String()
}
