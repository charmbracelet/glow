package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/boba"
	"github.com/charmbracelet/boba/spinner"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/muesli/reflow/indent"
	te "github.com/muesli/termenv"
)

// MSG

type stashErrMsg error

type gotStashMsg []*charm.Markdown

type stashSpinnerTickMsg struct{}

// MODEL

type stashState int

const (
	stashStateInit stashState = iota
	stashStateLoaded
)

type stashModel struct {
	cc        *charm.Client
	err       error
	state     stashState
	documents []*charm.Markdown
	page      int
	spinner   spinner.Model
}

// INIT

func stashInit(cc *charm.Client) (stashModel, boba.Cmd) {
	s := spinner.NewModel()
	s.Type = spinner.Dot
	s.ForegroundColor = common.SpinnerColor
	s.CustomMsgFunc = newSpinnerTickMsg

	m := stashModel{
		cc:      cc,
		spinner: s,
		page:    1,
	}

	return m, boba.Batch(
		getStash(m),
		spinner.Tick(s),
	)
}
func newSpinnerTickMsg() boba.Msg {
	return stashSpinnerTickMsg{}
}

// UPDATE

func stashUpdate(msg boba.Msg, m stashModel) (stashModel, boba.Cmd) {
	switch msg := msg.(type) {

	case stashErrMsg:
		m.err = msg

	case gotStashMsg:
		sort.Sort(charm.MarkdownsByCreatedAt(msg)) // sort by date
		m.documents = msg
		m.state = stashStateLoaded

	case stashSpinnerTickMsg:
		if m.state == stashStateInit {
			var cmd boba.Cmd
			m.spinner, cmd = spinner.Update(msg, m.spinner)
			return m, cmd
		}
	}

	return m, nil
}

// VIEW

func stashView(m stashModel) string {
	var s string
	switch m.state {
	case stashStateInit:
		s += spinner.View(m.spinner) + " Loading stash..."
	case stashStateLoaded:
		if len(m.documents) == 0 {
			s += stashEmtpyView(m)
			break
		}
		s += stashPopulatedView(m)
	}
	return "\n" + indent.String(s, 2)
}

func stashEmtpyView(m stashModel) string {
	return "Nothing stashed yet."
}

func stashPopulatedView(m stashModel) string {
	s := "Here's your markdown stash:\n\n"
	for _, v := range m.documents {
		s += stashListItemView(*v).renderNormal() + "\n\n"
	}
	s = strings.TrimSpace(s)
	return s
}

type stashListItemView charm.Markdown

func (m stashListItemView) renderNormal() string {
	line := common.VerticalLine(common.StateNormal) + " "
	var s string
	s += fmt.Sprintf("%s#%d%s\n", line, m.ID, m.title())
	s += fmt.Sprintf("%sStashed: %s", line, m.date())
	return s
}

func (m stashListItemView) date() string {
	s := m.CreatedAt.Format("02 Jan 2006 15:04:05 MST")
	return te.String(s).Foreground(common.Indigo.Color()).String()
}

func (m stashListItemView) title() string {
	if m.Note == "" {
		return ""
	}
	return ": " + te.String(m.Note).Foreground(common.Indigo.Color()).String()
}

// CMD

func getStash(m stashModel) boba.Cmd {
	return func() boba.Msg {
		stash, err := m.cc.GetStash(m.page)
		if err != nil {
			return stashErrMsg(err)
		}
		return gotStashMsg(stash)
	}
}
