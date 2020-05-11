package ui

import (
	"errors"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/charm/ui/keygen"
	"github.com/charmbracelet/tea"
	"github.com/charmbracelet/teaparty/spinner"
)

// NewProgram returns a new Tea program
func NewProgram() *tea.Program {
	return tea.NewProgram(initialize, update, view, subscriptions)
}

// MESSAGES

type fatalErrMsg error
type errMsg error
type newCharmClientMsg *charm.Client
type sshAuthErrMsg struct{}

// MODEL

type state int

const (
	stateInitCharmClient state = iota
	stateKeygenRunning
	stateKeygenFinished
	stateReady
)

type model struct {
	cc      *charm.Client
	user    *charm.User
	spinner spinner.Model
	keygen  keygen.Model
	state   state
	err     error
}

// INIT

func initialize() (tea.Model, tea.Cmd) {
	s := spinner.NewModel()
	s.Type = spinner.Dot
	s.ForegroundColor = common.SpinnerColor

	return model{
		spinner: s,
		state:   stateInitCharmClient,
	}, newCharmClient
}

// UPDATE

func update(msg tea.Msg, mdl tea.Model) (tea.Model, tea.Cmd) {
	m, ok := mdl.(model)
	if !ok {
		return model{err: errors.New("could not perform assertion on model in update")}, tea.Quit
	}

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		default:
			return m, nil
		}

	case fatalErrMsg:
		m.err = msg
		return m, tea.Quit

	case errMsg:
		m.err = msg
		return m, nil

	case sshAuthErrMsg:
		// If we haven't run the keygen yet, do that
		if m.state != stateKeygenFinished {
			m.state = stateKeygenRunning
			m.keygen = keygen.NewModel()
			return m, keygen.GenerateKeys
		}

		// The keygen didn't work
		m.err = errors.New("SSH authentication failed")
		return m, tea.Quit

	case spinner.TickMsg:
		m.spinner, _ = spinner.Update(msg, m.spinner)
		return m, nil

	case keygen.DoneMsg:
		m.state = stateKeygenFinished
		return m, newCharmClient

	case newCharmClientMsg:
		m.cc = msg
		m.state = stateReady
		return m, nil

	default:
		switch m.state {
		case stateKeygenRunning:
			mdl, cmd := keygen.Update(msg, tea.Model(m.keygen))
			keygenModel, ok := mdl.(keygen.Model)
			if !ok {
				m.err = errors.New("could not perform assertion on keygen model in main update")
				return m, tea.Quit
			}
			m.keygen = keygenModel
			return m, cmd
		}
		return m, nil
	}
}

// VIEW

func view(mdl tea.Model) string {
	m, ok := mdl.(model)
	if !ok {
		return "could not perform assertion on model in view"
	}

	if m.err != nil {
		return m.err.Error()
	}

	var s string
	switch m.state {
	case stateInitCharmClient:
		s += spinner.View(m.spinner) + " Initializing..."
	case stateKeygenRunning:
		s += keygen.View(m.keygen)
	case stateKeygenFinished:
		s += spinner.View(m.spinner) + " Re-initializing..."
	case stateReady:
		s += "Ready."
	}
	return s
}

// SUBSCRIPTIONS

func subscriptions(mdl tea.Model) tea.Subs {
	m, ok := mdl.(model)
	if !ok {
		return nil
	}

	subs := make(tea.Subs)

	switch m.state {
	case stateInitCharmClient:
		fallthrough
	case stateKeygenFinished:
		sub, err := spinner.MakeSub(m.spinner)
		if err == nil {
			subs["glow-spin"] = sub
		}
	case stateKeygenRunning:
		sub, err := keygen.Spin(m.keygen)
		if err == nil {
			subs["keygen-spin"] = sub
		}
	}

	return subs
}

// COMMANDS

func newCharmClient() tea.Msg {
	cfg, err := charm.ConfigFromEnv()
	if err != nil {
		return fatalErrMsg(err)
	}

	cc, err := charm.NewClient(cfg)
	if err == charm.ErrMissingSSHAuth {
		return sshAuthErrMsg{}
	} else if err != nil {
		return fatalErrMsg(err)
	}

	return newCharmClientMsg(cc)
}
