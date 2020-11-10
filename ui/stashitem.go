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
	normalFg    = makeFgStyle(common.NewColorPair("#dddddd", "#1a1a1a"))
	dimNormalFg = makeFgStyle(common.NewColorPair("#777777", "#A49FA5"))

	warmGrayFg    = makeFgStyle(common.NewColorPair("#979797", "#847A85"))
	dimWarmGrayFg = makeFgStyle(common.NewColorPair("#4D4D4D", "#C2B8C2"))

	grayFg    = makeFgStyle(common.NewColorPair("#626262", "#000"))
	dimGrayFg = makeFgStyle(common.NewColorPair("#3F3F3F", "#000"))

	greenFg    = makeFgStyle(common.NewColorPair("#04B575", "#04B575"))
	dimGreenFg = makeFgStyle(common.NewColorPair("#0B5137", "#82E1BF"))

	fuchsiaFg    = makeFgStyle(common.Fuschia)
	dimFuchsiaFg = makeFgStyle(common.NewColorPair("#99519E", "#F1A8FF"))

	dullFuchsiaFg    = makeFgStyle(common.NewColorPair("#AD58B4", "#F793FF"))
	dimDullFuchsiaFg = makeFgStyle(common.NewColorPair("#6B3A6F", "#F6C9FF"))

	indigoFg    = makeFgStyle(common.Indigo)
	dimIndigoFg = makeFgStyle(common.NewColorPair("#494690", "#9498FF"))

	subtleIndigoFg    = makeFgStyle(common.NewColorPair("#514DC1", "#7D79F6"))
	dimSubtleIndigoFg = makeFgStyle(common.NewColorPair("#383584", "#BBBDFF"))

	yellowFg     = makeFgStyle(common.YellowGreen)                        // renders light green on light backgrounds
	dullYellowFg = makeFgStyle(common.NewColorPair("#9BA92F", "#6BCB94")) // renders light green on light backgrounds
	redFg        = makeFgStyle(common.Red)
	faintRedFg   = makeFgStyle(common.FaintRed)
)

func makeFgStyle(c common.ColorPair) func(string) string {
	return te.Style{}.Foreground(c.Color()).Styled
}

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
			title = textinput.View(m.noteInput)
			date = dullYellowFg(date)
		case stashStateSearchNotes:
			if len(m.getNotes()) != 1 && m.searchInput.Value() == "" {
				gutter = dimDullFuchsiaFg(verticalLine)
				icon = dimDullFuchsiaFg(icon)
				title = dimFuchsiaFg(title)
				date = dimDullFuchsiaFg(date)
				break
			}
			// If we've filtered down to exactly item color it as though it's
			// not filtered, since pressing return will open it.
			fallthrough
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
			if m.state == stashStateSearchNotes && m.searchInput.Value() == "" {
				title = dimIndigoFg(title)
				date = dimSubtleIndigoFg(date)
			} else {
				title = indigoFg(title)
				date = subtleIndigoFg(date)
			}
		} else if m.state == stashStateSearchNotes && m.searchInput.Value() == "" {
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
