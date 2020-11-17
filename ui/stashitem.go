package ui

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/charm/ui/common"
	rw "github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"
	"github.com/sahilm/fuzzy"
)

const (
	newsPrefix           = "News: "
	verticalLine         = "│"
	noMemoTitle          = "No Memo"
	fileListingStashIcon = "• "
)

func stashItemView(b *strings.Builder, m stashModel, index int, md *markdown) {
	var (
		truncateTo = m.terminalWidth - stashViewHorizontalPadding*2
		gutter     string
		title      = md.Note
		date       = relativeTime(md.CreatedAt)
		icon       = ""
	)

	switch md.markdownType {
	case newsMarkdown:
		if title == "" {
			title = "News"
		} else {
			title = newsPrefix + truncate(title, truncateTo-rw.StringWidth(newsPrefix))
		}
	case stashedMarkdown, convertedMarkdown:
		icon = fileListingStashIcon
		if title == "" {
			title = noMemoTitle
		}
		title = truncate(title, truncateTo-rw.StringWidth(icon))
	default:
		title = truncate(title, truncateTo)
	}

	isSelected := index == m.index
	isFilteringNotes := m.state == stashStateFilterNotes

	// If there are multiple items being filtered we don't highlight a selected
	// item in the results. If we've filtered down to one item, however,
	// highlight that first item since pressing return will open it.
	singleFilteredItem := m.state == stashStateFilterNotes && len(m.getNotes()) == 1

	if isSelected && !isFilteringNotes || singleFilteredItem {
		// Selected item

		switch m.state {
		case stashStatePromptDelete:
			gutter = faintRedFg(verticalLine)
			icon = faintRedFg(icon)
			title = redFg(title)
			date = faintRedFg(date)
		case stashStateSettingNote:
			gutter = dullYellowFg(verticalLine)
			icon = ""
			title = m.noteInput.View()
			date = dullYellowFg(date)
		default:
			gutter = dullFuchsiaFg(verticalLine)
			icon = dullFuchsiaFg(icon)
			if m.state == stashStateShowFiltered || singleFilteredItem {
				s := termenv.Style{}.Foreground(common.Fuschia.Color())
				title = styleFilteredText(title, m.filterInput.Value(), s, s.Underline())
			} else {
				title = fuchsiaFg(title)
			}
			date = dullFuchsiaFg(date)
		}
	} else {
		// Regular (non-selected) items

		if md.markdownType == newsMarkdown {
			gutter = " "

			if isFilteringNotes && m.filterInput.Value() == "" {
				title = dimIndigoFg(title)
				date = dimSubtleIndigoFg(date)
			} else {
				s := termenv.Style{}.Foreground(common.Indigo.Color())
				title = styleFilteredText(title, m.filterInput.Value(), s, s.Underline())
				date = subtleIndigoFg(date)
			}
		} else if isFilteringNotes && m.filterInput.Value() == "" {
			icon = dimGreenFg(icon)
			if title == noMemoTitle {
				title = dimWarmGrayFg(title)
			} else {
				title = dimNormalFg(title)
			}
			gutter = " "
			date = dimWarmGrayFg(date)

		} else {

			icon = greenFg(icon)
			if title == noMemoTitle {
				title = warmGrayFg(title)
			} else {
				s := termenv.Style{}.Foreground(common.NewColorPair("#dddddd", "#1a1a1a").Color())
				title = styleFilteredText(title, m.filterInput.Value(), s, s.Underline())
			}
			gutter = " "
			date = warmGrayFg(date)
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
