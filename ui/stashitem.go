package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/boba/textinput"
	"github.com/charmbracelet/charm/ui/common"
	te "github.com/muesli/termenv"
)

const (
	newsPrefix   = "News: "
	verticalLine = "│"
	noMemoTitle  = "No Memo"
)

var (
	faintGreen   = common.NewColorPair("#2B4A3F", "#ABE5D1")
	green        = common.NewColorPair("#04B575", "#04B575")
	dullFuchsia  = common.NewColorPair("#AD58B4", "#F793FF")
	dullYellow   = common.NewColorPair("#9BA92F", "#6BCB94") // renders light green on light backgrounds
	warmGray     = common.NewColorPair("#979797", "#847A85")
	subtleIndigo = common.NewColorPair("#514DC1", "#7D79F6")
)

func stashItemView(b *strings.Builder, m stashModel, index int, md *markdown) *strings.Builder {

	truncateTo := m.terminalWidth - stashViewHorizontalPadding*2
	gutter := " "
	title := md.Note
	date := relativeTime(*md.CreatedAt)

	if md.markdownType == newsMarkdown {
		if title == "" {
			title = "News"
		} else {
			title = newsPrefix + truncate(title, truncateTo-len(newsPrefix))
		}
	} else {
		if title == "" {
			title = noMemoTitle
		}
		title = truncate(title, truncateTo)
	}

	if index == m.index {
		switch m.state {
		case stashStatePromptDelete:
			// Deleting
			gutter = te.String(verticalLine).Foreground(common.FaintRed.Color()).String()
			title = te.String(title).Foreground(common.Red.Color()).String()
			date = te.String(date).Foreground(common.FaintRed.Color()).String()
		case stashStateSettingNote:
			// Setting note
			gutter = te.String(verticalLine).Foreground(dullYellow.Color()).String()
			title = textinput.View(m.noteInput)
			date = te.String(date).Foreground(dullYellow.Color()).String()
		default:
			// Selected
			gutter = te.String(verticalLine).Foreground(dullFuchsia.Color()).String()
			title = te.String(title).Foreground(common.Fuschia.Color()).String()
			date = te.String(date).Foreground(dullFuchsia.Color()).String()
		}
	} else {
		// Normal
		if md.markdownType == newsMarkdown {
			gutter = " "
			title = te.String(title).Foreground(common.Indigo.Color()).String()
			date = te.String(date).Foreground(subtleIndigo.Color()).String()
		} else {
			if title == noMemoTitle {
				title = te.String(title).Foreground(warmGray.Color()).String()
			}
			gutter = " "
			date = te.String(date).Foreground(warmGray.Color()).String()
		}
	}

	fmt.Fprintf(b, "%s %s\n", gutter, title)
	fmt.Fprintf(b, "%s %s", gutter, date)

	return b
}
