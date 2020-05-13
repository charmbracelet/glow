package ui

import (
	"github.com/charmbracelet/boba"
	"github.com/charmbracelet/boba/spinner"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
)

// MSG

type stashErrMsg error

type gotStashMsg []*charm.Markdown

type stashSpinnerTickMsg struct{}

// MODEL

type stashState int

const (
	stashStateInit stashState = iota
	stashStateStashLoaded
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
		m.documents = msg
		m.state = stashStateStashLoaded

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
	if (m.state & stashStateStashLoaded) != 0 {
		return "Loaded stash."
	}
	s += spinner.View(m.spinner) + " Loading stash..."
	return s + "\n"
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
