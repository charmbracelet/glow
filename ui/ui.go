package ui

import (
	"errors"

	"github.com/charmbracelet/boba"
	"github.com/charmbracelet/boba/spinner"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/charm/ui/keygen"
	"github.com/muesli/reflow/indent"
	te "github.com/muesli/termenv"
)

var (
	glowLogoTextColor = common.Color("#ECFD65")
	statusBarBg       = common.NewColorPair("#242424", "#E6E6E6")
	statusBarFg       = common.NewColorPair("#5A5A5A", "#949494")
)

// NewProgram returns a new Boba program
func NewProgram(style string) *boba.Program {
	return boba.NewProgram(initialize(style), update, view)
}

// MESSAGES

type fatalErrMsg error
type errMsg error
type newCharmClientMsg *charm.Client
type sshAuthErrMsg struct{}
type terminalResizedMsg struct{}

type terminalSizeMsg struct {
	width  int
	height int
	err    error
}

func (t terminalSizeMsg) Size() (int, int) { return t.width, t.height }
func (t terminalSizeMsg) Error() error     { return t.err }

// MODEL

type state int

const (
	stateInitCharmClient state = iota
	stateKeygenRunning
	stateKeygenFinished
	stateShowStash
	stateShowDocument
)

type model struct {
	cc             *charm.Client
	user           *charm.User
	spinner        spinner.Model
	keygen         keygen.Model
	state          state
	err            error
	stash          stashModel
	pager          pagerModel
	terminalWidth  int
	terminalHeight int
}

func (m *model) unloadDocument() {
	m.state = stateShowStash
	m.stash.state = stashStateReady
	m.pager.unload()
}

// INIT

func initialize(style string) func() (boba.Model, boba.Cmd) {
	return func() (boba.Model, boba.Cmd) {
		s := spinner.NewModel()
		s.Type = spinner.Dot
		s.ForegroundColor = common.SpinnerColor

		if style == "auto" {
			dbg := te.HasDarkBackground()
			if dbg == true {
				style = "dark"
			} else {
				style = "light"
			}
		}

		return model{
				spinner: s,
				pager:   newPagerModel(style),
				state:   stateInitCharmClient,
			}, boba.Batch(
				newCharmClient,
				spinner.Tick(s),
				getTerminalSize(),
				listenForTerminalResize(),
			)
	}
}

// UPDATE

func update(msg boba.Msg, mdl boba.Model) (boba.Model, boba.Cmd) {
	m, ok := mdl.(model)
	if !ok {
		return model{err: errors.New("could not perform assertion on model in update")}, boba.Quit
	}

	var (
		cmd  boba.Cmd
		cmds []boba.Cmd
	)

	switch msg := msg.(type) {

	case boba.KeyMsg:
		switch msg.String() {
		case "q":
			fallthrough
		case "esc":
			if m.state == stateShowDocument {
				var cmd boba.Cmd
				if m.pager.state == pagerStateBrowse {
					// Exit pager
					m.unloadDocument()
				} else {
					// Pass message through to pager
					m.pager, cmd = pagerUpdate(msg, m.pager)
				}
				return m, cmd
			}
			return m, boba.Quit

		case "ctrl+c":
			return m, boba.Quit

		// Repaint
		case "ctrl+l":
			return m, getTerminalSize()
		}

	case fatalErrMsg:
		m.err = msg
		return m, boba.Quit

	case errMsg:
		m.err = msg
		return m, nil

	case terminalResizedMsg:
		cmds = append(cmds,
			getTerminalSize(),
			listenForTerminalResize(),
		)

	case terminalSizeMsg:
		if msg.Error() != nil {
			m.err = msg.Error()
		}
		w, h := msg.Size()
		m.terminalWidth = w
		m.terminalHeight = h
		m.stash.setSize(w, h)
		m.pager.setSize(w, h)

		// TODO: load more stash pages if we've resized, are on the last page,
		// and haven't loaded more pages yet.

	case sshAuthErrMsg:
		// If we haven't run the keygen yet, do that
		if m.state != stateKeygenFinished {
			m.state = stateKeygenRunning
			m.keygen = keygen.NewModel()
			cmds = append(cmds, keygen.GenerateKeys)
		} else {
			// The keygen didn't work and we can't auth
			m.err = errors.New("SSH authentication failed")
			return m, boba.Quit
		}

	case spinner.TickMsg:
		switch m.state {
		case stateInitCharmClient:
			m.spinner, cmd = spinner.Update(msg, m.spinner)
		}
		cmds = append(cmds, cmd)

	case keygen.DoneMsg:
		m.state = stateKeygenFinished
		cmds = append(cmds, newCharmClient)

	case noteSavedMsg:
		// A note was saved to a document. This will have be done in the
		// pager, so we'll need to find the corresponding note in the stash.
		// So, pass the message to the stash for processing.
		m.stash, cmd = stashUpdate(msg, m.stash)
		cmds = append(cmds, cmd)

	case newCharmClientMsg:
		m.cc = msg
		m.state = stateShowStash
		m.stash, cmd = stashInit(msg)
		m.stash.setSize(m.terminalWidth, m.terminalHeight)
		m.pager.cc = msg
		cmds = append(cmds, cmd)

	case gotStashedItemMsg:
		// Loaded markdown document from the server. We'll render it before
		// loading it into the pager.
		m.pager.currentDocument = msg
		cmds = append(cmds, renderWithGlamour(m.pager, msg.Body))

	case contentRenderedMsg:
		m.state = stateShowDocument

	}

	switch m.state {

	case stateKeygenRunning:
		// Process keygen
		mdl, cmd := keygen.Update(msg, boba.Model(m.keygen))
		keygenModel, ok := mdl.(keygen.Model)
		if !ok {
			m.err = errors.New("could not perform assertion on keygen model in main update")
			return m, boba.Quit
		}
		m.keygen = keygenModel
		cmds = append(cmds, cmd)

	case stateShowStash:
		// Process stash
		m.stash, cmd = stashUpdate(msg, m.stash)
		cmds = append(cmds, cmd)

	case stateShowDocument:
		// Process pager
		m.pager, cmd = pagerUpdate(msg, m.pager)
		cmds = append(cmds, cmd)
	}

	return m, boba.Batch(cmds...)
}

// VIEW

func view(mdl boba.Model) string {
	m, ok := mdl.(model)
	if !ok {
		return "could not perform assertion on model in view"
	}

	if m.err != nil {
		return m.err.Error() + "\n"
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
		return stashView(m.stash)
	case stateShowDocument:
		return pagerView(m.pager)
	}

	return "\n" + indent.String(s, 2)
}

// COMMANDS

func listenForTerminalResize() boba.Cmd {
	return boba.OnResize(func() boba.Msg {
		return terminalResizedMsg{}
	})
}

func getTerminalSize() boba.Cmd {
	return boba.GetTerminalSize(func(w, h int, err error) boba.TerminalSizeMsg {
		return terminalSizeMsg{width: w, height: h, err: err}
	})
}

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

// ETC

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
