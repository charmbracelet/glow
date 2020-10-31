package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/charm/ui/common"
	rw "github.com/mattn/go-runewidth"
	te "github.com/muesli/termenv"
)

const (
	newsPrefix           = "News: "
	verticalLine         = "│"
	noMemoTitle          = "No Memo"
	fileListingStashIcon = "• "
)

var (
	greenFg        = te.Style{}.Foreground(common.NewColorPair("#04B575", "#04B575").Color()).Styled
	fuchsiaFg      = te.Style{}.Foreground(common.Fuschia.Color()).Styled
	dullFuchsiaFg  = te.Style{}.Foreground(common.NewColorPair("#AD58B4", "#F793FF").Color()).Styled
	yellowFg       = te.Style{}.Foreground(common.YellowGreen.Color()).Styled                        // renders light green on light backgrounds
	dullYellowFg   = te.Style{}.Foreground(common.NewColorPair("#9BA92F", "#6BCB94").Color()).Styled // renders light green on light backgrounds
	subtleIndigoFg = te.Style{}.Foreground(common.NewColorPair("#514DC1", "#7D79F6").Color()).Styled
	redFg          = te.Style{}.Foreground(common.Red.Color()).Styled
	faintRedFg     = te.Style{}.Foreground(common.FaintRed.Color()).Styled
	warmGrayFg     = te.Style{}.Foreground(common.NewColorPair("#979797", "#847A85").Color()).Styled
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

	if index == m.index {
		switch m.state {
		case stashStatePromptDelete:
			// Deleting
			gutter = faintRedFg(verticalLine)
			icon = faintRedFg(icon)
			title = redFg(title)
			date = faintRedFg(date)
		case stashStateSettingNote:
			// Setting note
			gutter = dullYellowFg(verticalLine)
			icon = ""
			title = textinput.View(m.noteInput)
			date = dullYellowFg(date)
		case stashStateSearchNotes:
			if len(m.getNotes()) != 1 {
				gutter = dullFuchsiaFg(verticalLine)
				icon = dullFuchsiaFg(icon)
				title = dullFuchsiaFg(title)
				break
			}
			fallthrough
		default:
			// Selected
			gutter = dullFuchsiaFg(verticalLine)
			icon = dullFuchsiaFg(icon)
			title = fuchsiaFg(title)
			date = dullFuchsiaFg(date)
		}
	} else {
		// Normal
		if md.markdownType == newsMarkdown {
			gutter = " "
			if m.state == stashStateSearchNotes {
				title = subtleIndigoFg(title)
			} else {
				title = te.String(title).Foreground(common.Indigo.Color()).String()
			}
			date = subtleIndigoFg(date)
		} else {
			icon = greenFg(icon)
			if title == noMemoTitle {
				title = warmGrayFg(title)
			}
			gutter = " "
			date = warmGrayFg(date)
		}

		if m.state == stashStateSearchNotes {
			icon = common.Subtle(icon)
			title = common.Subtle(title)
			gutter = common.Subtle(gutter)
			date = common.Subtle(date)
		}
	}

	fmt.Fprintf(b, "%s %s%s\n", gutter, icon, title)
	fmt.Fprintf(b, "%s %s", gutter, date)
}
