package ui

import (
	"fmt"
	"strings"

	rw "github.com/mattn/go-runewidth"
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
	notFilteringNotes := m.state != stashStateFilterNotes

	// If there are multiple items being filtered we don't highlight a selected
	// item in the results. If we've filtered down to one item, however,
	// highlight that first item since pressing return will open it.
	singleFilteredItem := m.state == stashStateFilterNotes && len(m.getNotes()) == 1

	if isSelected && notFilteringNotes || singleFilteredItem {
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
			title = fuchsiaFg(title)
			date = dullFuchsiaFg(date)
		}
	} else {
		// Regular (non-selected) items

		if md.markdownType == newsMarkdown {
			gutter = " "
			if m.state == stashStateFilterNotes && m.filterInput.Value() == "" {
				title = dimIndigoFg(title)
				date = dimSubtleIndigoFg(date)
			} else {
				title = indigoFg(title)
				date = subtleIndigoFg(date)
			}
		} else if m.state == stashStateFilterNotes && m.filterInput.Value() == "" {
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
				title = normalFg(title)
			}
			gutter = " "
			date = warmGrayFg(date)
		}

	}

	fmt.Fprintf(b, "%s %s%s\n", gutter, icon, title)
	fmt.Fprintf(b, "%s %s", gutter, date)
}
