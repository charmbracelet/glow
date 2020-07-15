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
	newsPrefix   = "News: "
	verticalLine = "│"
	noMemoTitle  = "No Memo"
	stashIcon    = "• "
)

var (
	greenFg        = te.Style{}.Foreground(common.NewColorPair("#04B575", "#04B575").Color()).Styled
	faintGreenFg   = te.Style{}.Foreground(common.NewColorPair("#2B4A3F", "#ABE5D1").Color()).Styled
	fuchsiaFg      = te.Style{}.Foreground(common.Fuschia.Color()).Styled
	dullFuchsiaFg  = te.Style{}.Foreground(common.NewColorPair("#AD58B4", "#F793FF").Color()).Styled
	yellowFg       = te.Style{}.Foreground(common.YellowGreen.Color()).Styled                        // renders light green on light backgrounds
	dullYellowFg   = te.Style{}.Foreground(common.NewColorPair("#9BA92F", "#6BCB94").Color()).Styled // renders light green on light backgrounds
	indigoFg       = te.Style{}.Foreground(common.Indigo.Color()).Styled
	subtleIndigoFg = te.Style{}.Foreground(common.NewColorPair("#514DC1", "#7D79F6").Color()).Styled
	redFg          = te.Style{}.Foreground(common.Red.Color()).Styled
	faintRedFg     = te.Style{}.Foreground(common.FaintRed.Color()).Styled
	warmGrayFg     = te.Style{}.Foreground(common.NewColorPair("#979797", "#847A85").Color()).Styled
)

func stashItemView(b *strings.Builder, m stashModel, index int, md *markdown) {

	truncateTo := m.terminalWidth - stashViewHorizontalPadding*2
	gutter := " "
	title := md.Note
	date := relativeTime(*md.CreatedAt)
	icon := ""

	switch md.markdownType {
	case newsMarkdown:
		if title == "" {
			title = "News"
		} else {
			title = newsPrefix + truncate(title, truncateTo-rw.StringWidth(newsPrefix))
		}
	case stashedMarkdown:
		icon = stashIcon
		icon = stashIcon
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
			title = te.String(title).Foreground(common.Indigo.Color()).String()
			date = subtleIndigoFg(date)
		} else {
			icon = greenFg(icon)
			if title == noMemoTitle {
				title = warmGrayFg(title)
			}
			gutter = " "
			date = warmGrayFg(date)
		}
	}

	fmt.Fprintf(b, "%s %s%s\n", gutter, icon, title)
	fmt.Fprintf(b, "%s %s", gutter, date)
}
