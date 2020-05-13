package ui

import (
	"errors"

	"github.com/charmbracelet/boba"
	"github.com/charmbracelet/boba/spinner"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/charm/ui/keygen"
)

// NewProgram returns a new Boba program
func NewProgram() *boba.Program {
	return boba.NewProgram(initialize, update, view)
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
	stateShowStash
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

func initialize() (boba.Model, boba.Cmd) {
	s := spinner.NewModel()
	s.Type = spinner.Dot
	s.ForegroundColor = common.SpinnerColor

	return model{
			spinner: s,
			state:   stateInitCharmClient,
		}, boba.Batch(
			newCharmClient,
			spinner.Tick(s),
		)
}

// UPDATE

func update(msg boba.Msg, mdl boba.Model) (boba.Model, boba.Cmd) {
	m, ok := mdl.(model)
	if !ok {
		return model{err: errors.New("could not perform assertion on model in update")}, boba.Quit
	}

	switch msg := msg.(type) {

	case boba.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, boba.Quit
		default:
			return m, nil
		}

	case fatalErrMsg:
		m.err = msg
		return m, boba.Quit

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
		return m, boba.Quit

	case spinner.TickMsg:
		if m.state == stateInitCharmClient {
			var cmd boba.Cmd
			m.spinner, cmd = spinner.Update(msg, m.spinner)
			return m, cmd
		}

	case keygen.DoneMsg:
		m.state = stateKeygenFinished
		return m, newCharmClient

	case newCharmClientMsg:
		m.cc = msg
		m.state = stateShowStash
		return m, nil

	default:
		switch m.state {
		case stateKeygenRunning:
			mdl, cmd := keygen.Update(msg, boba.Model(m.keygen))
			keygenModel, ok := mdl.(keygen.Model)
			if !ok {
				m.err = errors.New("could not perform assertion on keygen model in main update")
				return m, boba.Quit
			}
			m.keygen = keygenModel
			return m, cmd
		}
		return m, nil
	}

	return m, nil
}

// VIEW

func view(mdl boba.Model) string {
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
	case stateShowStash:
		s += "Ready."
	}
	return s + "\n"
}

// COMMANDS

func newCharmClient() boba.Msg {
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
