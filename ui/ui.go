package ui

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/boba"
	"github.com/charmbracelet/boba/pager"
	"github.com/charmbracelet/boba/spinner"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/charm/ui/keygen"
	"github.com/charmbracelet/glamour"
	"github.com/muesli/reflow/indent"
	te "github.com/muesli/termenv"
)

const (
	statusBarHeight = 1
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
type contentRenderedMsg string

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
	style          string // style to use
	cc             *charm.Client
	user           *charm.User
	spinner        spinner.Model
	keygen         keygen.Model
	state          state
	err            error
	stash          stashModel
	pager          pager.Model
	terminalWidth  int
	terminalHeight int
	docNote        string
}

func (m *model) unloadDocument() {
	m.pager = pager.Model{}
	m.state = stateShowStash
	m.stash.state = stashStateStashLoaded
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
				style:   style,
				spinner: s,
				state:   stateInitCharmClient,
			}, boba.Batch(
				newCharmClient,
				spinner.Tick(s),
				getTerminalSize(),
			)
	}
}

// UPDATE

func update(msg boba.Msg, mdl boba.Model) (boba.Model, boba.Cmd) {
	m, ok := mdl.(model)
	if !ok {
		return model{err: errors.New("could not perform assertion on model in update")}, boba.Quit
	}

	var cmd boba.Cmd

	switch msg := msg.(type) {

	case boba.KeyMsg:
		switch msg.String() {

		case "q":
			fallthrough
		case "esc":
			if m.state == stateShowDocument {
				m.unloadDocument()
				return m, nil
			}
			return m, boba.Quit

		case "ctrl+c":
			return m, boba.Quit

		// Re-render
		case "ctrl+l":
			return m, getTerminalSize()
		}

	case fatalErrMsg:
		m.err = msg
		return m, boba.Quit

	case errMsg:
		m.err = msg
		return m, nil

	case terminalSizeMsg:
		if msg.Error() != nil {
			m.err = msg.Error()
			return m, nil
		}
		w, h := msg.Size()
		m.terminalWidth = w
		m.terminalHeight = h
		m.stash.SetSize(w, h)
		// TODO: load more stash pages if we've resized, are on the last page,
		// and haven't loaded more pages yet.
		return m, nil

	case sshAuthErrMsg:
		// If we haven't run the keygen yet, do that
		if m.state != stateKeygenFinished {
			m.state = stateKeygenRunning
			m.keygen = keygen.NewModel()
			return m, keygen.GenerateKeys
		}

		// The keygen didn't work and we can't auth
		m.err = errors.New("SSH authentication failed")
		return m, boba.Quit

	case spinner.TickMsg:
		switch m.state {
		case stateInitCharmClient:
			m.spinner, cmd = spinner.Update(msg, m.spinner)
		}
		return m, cmd

	case stashSpinnerTickMsg:
		if m.state == stateShowStash {
			m.stash, cmd = stashUpdate(msg, m.stash)
		}
		return m, cmd

	case keygen.DoneMsg:
		m.state = stateKeygenFinished
		return m, newCharmClient

	case newCharmClientMsg:
		m.cc = msg
		m.state = stateShowStash
		m.stash, cmd = stashInit(m.cc)
		m.stash.SetSize(m.terminalWidth, m.terminalHeight)
		return m, cmd

	case gotStashedItemMsg:
		// We've received stashed item data. Render with Glamour and send to
		// the pager.
		m.pager = pager.NewModel(
			m.terminalWidth,
			m.terminalHeight-statusBarHeight,
		)

		m.docNote = msg.Note
		return m, renderWithGlamour(m, msg.Body)

	case contentRenderedMsg:
		m.state = stateShowDocument
		m.pager.SetContent(string(msg))
		return m, nil

	}

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

	case stateShowStash:
		m.stash, cmd = stashUpdate(msg, m.stash)
		return m, cmd

	case stateShowDocument:
		// Process keys (and eventually mouse) with pager.Update
		var cmd boba.Cmd
		m.pager, cmd = pager.Update(msg, m.pager)
		return m, cmd
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
		return fmt.Sprintf("\n%s\n%s", pager.View(m.pager), statusBarView(m))
	}

	return "\n" + indent.String(s, 2)
}

func glowLogoView(text string) string {
	return te.String(text).
		Bold().
		Foreground(glowLogoTextColor).
		Background(common.Fuschia.Color()).
		String()
}

func statusBarView(m model) string {
	// Logo
	logoText := " Glow "
	logo := glowLogoView(logoText)

	// Note
	noteText := m.docNote
	if len(noteText) == 0 {
		noteText = "(No title)"
	}
	noteText = " " + noteText
	note := te.String(noteText).
		Foreground(statusBarFg.Color()).
		Background(statusBarBg.Color()).String()

	// Scroll percent
	percentText := fmt.Sprintf(" %3.f%% ", m.pager.ScrollPercent()*100)
	percent := te.String(percentText).
		Foreground(statusBarFg.Color()).
		Background(statusBarBg.Color()).
		String()

	// Empty space
	emptyCell := te.String(" ").Background(statusBarBg.Color()).String()
	emptySpace := strings.Repeat(emptyCell, m.terminalWidth-len(logoText)-len(noteText)-len(percentText))

	return logo + note + emptySpace + percent
}

// COMMANDS
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

func renderWithGlamour(m model, md string) boba.Cmd {
	return func() boba.Msg {
		s, err := glamourRender(m, md)
		if err != nil {
			return errMsg(err)
		}
		return contentRenderedMsg(s)
	}
}

// ETC

// This is where the magic happens
func glamourRender(m model, markdown string) (string, error) {

	if os.Getenv("GLOW_DISABLE_GLAMOUR") != "" {
		return markdown, nil
	}

	// initialize glamour
	var gs glamour.TermRendererOption
	if m.style == "auto" {
		gs = glamour.WithAutoStyle()
	} else {
		gs = glamour.WithStylePath(m.style)
	}

	r, err := glamour.NewTermRenderer(
		gs,
		glamour.WithWordWrap(m.terminalWidth),
	)
	if err != nil {
		return "", err
	}

	out, err := r.Render(markdown)
	if err != nil {
		return "", err
	}

	// trim lines
	lines := strings.Split(string(out), "\n")

	var content string
	for i, s := range lines {
		content += strings.TrimSpace(s)

		// don't add an artificial newline after the last split
		if i+1 < len(lines) {
			content += "\n"
		}
	}

	return content, nil
}

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
