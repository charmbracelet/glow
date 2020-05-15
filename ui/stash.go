package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/boba"
	"github.com/charmbracelet/boba/paginator"
	"github.com/charmbracelet/boba/spinner"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/muesli/reflow/indent"
	te "github.com/muesli/termenv"
)

const (
	itemHeight    = 3
	topPadding    = 5
	bottomPadding = 4
)

// MSG

type stashErrMsg error
type stashSpinnerTickMsg struct{}
type gotStashMsg []*charm.Markdown
type gotStashedItemMsg *charm.Markdown

// MODEL

type stashState int

const (
	stashStateInit stashState = iota
	stashStateLoaded
	stashStateLoadingItem
)

type stashModel struct {
	cc             *charm.Client
	err            error
	state          stashState
	documents      []*charm.Markdown
	spinner        spinner.Model
	index          int
	terminalWidth  int
	terminalHeight int

	// This handles the local pagination, which is different than the page
	// we're fetching from on the server side
	paginator paginator.Model

	// Page we're fetching items from on the server, which is different from
	// the local pagination. Generally, the server will return more items than
	// we can display at a time so we can paginate locally without having to
	// fetch every time.
	page int
}

func (m *stashModel) SetSize(width, height int) {
	m.terminalWidth = width
	m.terminalHeight = height

	// Update the paginator
	perPage := (m.terminalHeight - topPadding - bottomPadding) / itemHeight
	m.paginator.PerPage = perPage
	m.paginator.SetTotalPages(len(m.documents))

	// Make sure the page stays in bounds
	if m.paginator.Page >= m.paginator.TotalPages-1 {
		m.paginator.Page = m.paginator.TotalPages - 1
	}
}

// INIT

func stashInit(cc *charm.Client) (stashModel, boba.Cmd) {
	s := spinner.NewModel()
	s.Type = spinner.Dot
	s.ForegroundColor = common.SpinnerColor
	s.CustomMsgFunc = func() boba.Msg { return stashSpinnerTickMsg{} }

	p := paginator.NewModel()
	p.Type = paginator.Dots
	p.InactiveDot = common.Subtle("•")

	m := stashModel{
		cc:        cc,
		spinner:   s,
		page:      1,
		paginator: p,
	}

	return m, boba.Batch(
		loadStash(m),
		spinner.Tick(s),
	)
}

// UPDATE

func stashUpdate(msg boba.Msg, m stashModel) (stashModel, boba.Cmd) {
	var (
		cmd  boba.Cmd
		cmds []boba.Cmd
	)

	switch msg := msg.(type) {

	case boba.KeyMsg:
		// Don't respond to keystrokes if we're still loading
		if m.state == stashStateInit {
			return m, nil
		}

		switch msg.String() {

		case "k":
			fallthrough
		case "up":
			m.index--
			if m.index < 0 && m.paginator.Page == 0 {
				// Stop
				m.index = 0
			} else if m.index < 0 {
				// Go to previous page
				m.paginator.PrevPage()
				m.index = m.paginator.ItemsOnPage(len(m.documents)) - 1
			}
			return m, nil

		case "j":
			fallthrough
		case "down":
			itemsOnPage := m.paginator.ItemsOnPage(len(m.documents))
			m.index++
			if m.index >= itemsOnPage && m.paginator.OnLastPage() {
				// Stop
				m.index = itemsOnPage - 1
			} else if m.index >= itemsOnPage {
				// Go to next page
				m.index = 0
				m.paginator.NextPage()
			}
			return m, nil

		case "enter":
			m.state = stashStateLoadingItem
			indexToLoad := m.paginator.Page*m.paginator.PerPage + m.index
			return m, boba.Batch(
				loadStashedItem(m.cc, m.documents[indexToLoad].ID),
				spinner.Tick(m.spinner),
			)

		}

	case stashErrMsg:
		m.err = msg

	case gotStashMsg:
		sort.Sort(charm.MarkdownsByCreatedAt(msg)) // sort by date
		m.documents = append(m.documents, msg...)
		m.state = stashStateLoaded
		m.paginator.SetTotalPages(len(m.documents))

	case stashSpinnerTickMsg:
		if m.state == stashStateInit || m.state == stashStateLoadingItem {
			m.spinner, cmd = spinner.Update(msg, m.spinner)
			return m, cmd
		}
	}

	if m.state == stashStateLoaded {
		m.paginator, cmd = paginator.Update(msg, m.paginator)
		cmds = append(cmds, cmd)
	}

	return m, boba.Batch(cmds...)
}

// VIEW

func stashView(m stashModel) string {
	var s string
	switch m.state {
	case stashStateInit:
		s += spinner.View(m.spinner) + " Loading stash..."
	case stashStateLoadingItem:
		s += spinner.View(m.spinner) + " Loading document..."
	case stashStateLoaded:
		if len(m.documents) == 0 {
			s += stashEmtpyView(m)
			break
		}

		// Blank lines we'll need to fill with newlines fo the viewport is
		// properly filled
		numBlankLines := (m.terminalHeight - topPadding - bottomPadding) % itemHeight
		blankLines := ""
		if numBlankLines > 0 {
			blankLines = strings.Repeat("\n", numBlankLines)
		}

		s += fmt.Sprintf(
			"%s\n\nHere’s your markdown stash.\n\n%s\n\n%s%s\n\n%s",
			glowLogoView(" Glow "),
			stashPopulatedView(m),
			blankLines,
			paginator.View(m.paginator),
			helpView(m),
		)
	}
	return "\n" + indent.String(s, 2)
}

func stashEmtpyView(m stashModel) string {
	return "Nothing stashed yet."
}

func stashPopulatedView(m stashModel) string {
	var s string

	start, end := m.paginator.GetSliceBounds(len(m.documents))
	docs := m.documents[start:end]

	for i, v := range docs {
		state := common.StateNormal
		if i == m.index {
			state = common.StateSelected
		}
		s += stashListItemView(*v).render(state) + "\n\n"
	}
	s = strings.TrimSpace(s) // trim final newlines

	// If there aren't enough items to fill up this page (always the last page)
	// then we need to add some newlines to fill up the space to push the
	// footer stuff down elsewhere.
	itemsOnPage := m.paginator.ItemsOnPage(len(m.documents))
	if itemsOnPage < m.paginator.PerPage {
		n := (m.paginator.PerPage - itemsOnPage) * itemHeight
		s += strings.Repeat("\n", n)
	}

	return s
}

func helpView(m stashModel) string {
	h := []string{"enter: open"}
	if len(m.documents) > 0 {
		h = append(h, "j/k, ↑/↓: choose")
	}
	if m.paginator.TotalPages > 1 {
		h = append(h, "h/l, ←/→: page")
	}
	h = append(h, []string{"x: delete", "esc: exit"}...)
	return common.HelpView(h...)
}

type stashListItemView charm.Markdown

func (m stashListItemView) render(state common.State) string {
	line := common.VerticalLine(state) + " "
	keyColor := common.NoColor
	switch state {
	case common.StateSelected:
		keyColor = common.Fuschia
	case common.StateDeleting:
		keyColor = common.Red
	}
	titleKey := strconv.Itoa(m.ID)
	if m.Note != "" {
		titleKey += ":"
	}
	titleKey = te.String("#" + titleKey).Foreground(keyColor.Color()).String()
	dateKey := te.String("Stashed:").Foreground(keyColor.Color()).String()
	var s string
	s += fmt.Sprintf("%s%s %s\n", line, titleKey, m.title(state))
	s += fmt.Sprintf("%s%s %s", line, dateKey, m.date(state))
	return s
}

func (m stashListItemView) date(state common.State) string {
	c := common.Indigo
	if state == common.StateDeleting {
		c = common.FaintRed
	}
	s := m.CreatedAt.Format("02 Jan 2006 15:04:05 MST")
	return te.String(s).Foreground(c.Color()).String()
}

func (m stashListItemView) title(state common.State) string {
	if m.Note == "" {
		return ""
	}
	c := common.Indigo
	if state == common.StateDeleting {
		c = common.Red
	}
	return te.String(m.Note).Foreground(c.Color()).String()
}

// CMD

func loadStash(m stashModel) boba.Cmd {
	return func() boba.Msg {
		stash, err := m.cc.GetStash(m.page)
		if err != nil {
			return stashErrMsg(err)
		}
		return gotStashMsg(stash)
	}
}

func loadStashedItem(cc *charm.Client, id int) boba.Cmd {
	return func() boba.Msg {
		m, err := cc.GetStashMarkdown(id)
		if err != nil {
			return stashErrMsg(err)
		}
		return gotStashedItemMsg(m)
	}
}
