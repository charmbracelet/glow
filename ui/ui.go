package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/charm/ui/keygen"
	te "github.com/muesli/termenv"
)

const (
	noteCharacterLimit = 256 // totally arbitrary
)

var (
	glowLogoTextColor = common.Color("#ECFD65")
)

// NewProgram returns a new Tea program
func NewProgram(style string) *tea.Program {
	return tea.NewProgram(initialize(style), update, view)
}

// MESSAGES

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
	stateShowDocument
)

// Stringn translates the staus to a human-readable string. This is just for
// debugging.
func (s state) String() string {
	return [...]string{
		"initializing",
		"running keygen",
		"keygen finished",
		"showing stash",
		"showing document",
	}[s]
}

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
	m.pager.showHelp = false
}

// INIT

func initialize(style string) func() (tea.Model, tea.Cmd) {
	return func() (tea.Model, tea.Cmd) {
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
			}, tea.Batch(
				newCharmClient,
				spinner.Tick(s),
			)
	}
}

// UPDATE

func update(msg tea.Msg, mdl tea.Model) (tea.Model, tea.Cmd) {
	m, ok := mdl.(model)
	if !ok {
		return model{
			err: errors.New("could not perform assertion on model in update"),
		}, tea.Quit
	}

	// If there's been an error, any key exits
	if m.err != nil {
		if _, ok := msg.(tea.KeyMsg); ok {
			return m, tea.Quit
		}
	}

	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			fallthrough
		case "esc":
			var cmd tea.Cmd

			switch m.state {
			case stateShowStash:

				switch m.stash.state {
				case stashStateSettingNote:
					fallthrough
				case stashStatePromptDelete:
					m.stash, cmd = stashUpdate(msg, m.stash)
					return m, cmd
				}

			case stateShowDocument:
				if m.pager.state == pagerStateBrowse {
					m.unloadDocument() // exits pager
					if m.pager.viewport.HighPerformanceRendering {
						cmd = tea.ClearScrollArea
					}
				} else {
					m.pager, cmd = pagerUpdate(msg, m.pager)
				}
				return m, cmd
			}

			return m, tea.Quit

		case "ctrl+c":
			return m, tea.Quit

		// Repaint
		case "ctrl+l":
			// TODO
			return m, nil
		}

	case errMsg:
		m.err = msg
		return m, nil

	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height
		m.stash.setSize(msg.Width, msg.Height)
		m.pager.setSize(msg.Width, msg.Height)

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
			return m, tea.Quit
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

	case fetchedMarkdownMsg:
		m.pager.currentDocument = msg
		cmds = append(cmds, renderWithGlamour(m.pager, msg.Body))

	case contentRenderedMsg:
		m.state = stateShowDocument

	}

	switch m.state {

	case stateKeygenRunning:
		// Process keygen
		mdl, cmd := keygen.Update(msg, tea.Model(m.keygen))
		keygenModel, ok := mdl.(keygen.Model)
		if !ok {
			m.err = errors.New("could not perform assertion on keygen model in main update")
			return m, tea.Quit
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

	return m, tea.Batch(cmds...)
}

// VIEW

func view(mdl tea.Model) string {

	m, ok := mdl.(model)
	if !ok {
		return "could not perform assertion on model in view"
	}

	if m.err != nil {
		return "\n" + indent(errorView(m.err), 2)
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

	return "\n" + indent(s, 2)
}

func errorView(err error) string {
	return fmt.Sprintf("%s\n\n%v\n\n%s",
		te.String(" ERROR ").
			Foreground(common.Cream.Color()).
			Background(common.Red.Color()).
			String(),
		err,
		common.Subtle("Press any key to exit"),
	)
}

// COMMANDS

func newCharmClient() tea.Msg {
	cfg, err := charm.ConfigFromEnv()
	if err != nil {
		return errMsg(err)
	}

	cc, err := charm.NewClient(cfg)
	if err == charm.ErrMissingSSHAuth {
		return sshAuthErrMsg{}
	} else if err != nil {
		return errMsg(err)
	}

	return newCharmClientMsg(cc)
}

func indent(s string, n int) string {
	if n <= 0 || s == "" {
		return s
	}
	l := strings.Split(s, "\n")
	b := strings.Builder{}
	i := strings.Repeat(" ", n)
	for _, v := range l {
		fmt.Fprintf(&b, "%s%s\n", i, v)
	}
	return b.String()
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
