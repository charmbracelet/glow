package ui

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/glamour"
	runewidth "github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/ansi"
	te "github.com/muesli/termenv"
)

const statusBarHeight = 1

var (
	pagerHelpHeight int

	mintGreen = common.NewColorPair("#89F0CB", "#89F0CB")
	darkGreen = common.NewColorPair("#1C8760", "#1C8760")

	noteHeading = te.String(" Set Memo ").
			Foreground(common.Cream.Color()).
			Background(common.Green.Color()).
			String()

	statusBarNoteFg = common.NewColorPair("#7D7D7D", "#656565")
	statusBarBg     = common.NewColorPair("#242424", "#E6E6E6")

	// Styling funcs
	statusBarScrollPosStyle        = newStyle(common.NewColorPair("#5A5A5A", "#949494"), statusBarBg)
	statusBarNoteStyle             = newStyle(statusBarNoteFg, statusBarBg)
	statusBarHelpStyle             = newStyle(statusBarNoteFg, common.NewColorPair("#323232", "#DCDCDC"))
	statusBarStashDotStyle         = newStyle(common.Green, statusBarBg)
	statusBarMessageStyle          = newStyle(mintGreen, darkGreen)
	statusBarMessageStashIconStyle = newStyle(mintGreen, darkGreen)
	statusBarMessageScrollPosStyle = newStyle(mintGreen, darkGreen)
	statusBarMessageHelpStyle      = newStyle(common.NewColorPair("#B6FFE4", "#B6FFE4"), common.Green)
	helpViewStyle                  = newStyle(statusBarNoteFg, common.NewColorPair("#1B1B1B", "#f2f2f2"))
)

type contentRenderedMsg string
type noteSavedMsg *charm.Markdown
type stashSuccessMsg markdown
type stashErrMsg struct{ err error }

func (s stashErrMsg) Error() string { return s.err.Error() }

type pagerState int

const (
	pagerStateBrowse pagerState = iota
	pagerStateSetNote
	pagerStateStashing
	pagerStateStashSuccess
	pagerStateStatusMessage
)

type pagerModel struct {
	general   *general
	viewport  viewport.Model
	state     pagerState
	showHelp  bool
	textInput textinput.Model
	spinner   spinner.Model

	statusMessage      string
	statusMessageTimer *time.Timer

	// Current document being rendered, sans-glamour rendering. We cache
	// it here so we can re-render it on resize.
	currentDocument markdown

	// Newly stashed markdown. We store it here temporarily so we can replace
	// currentDocument above after a stash.
	stashedDocument *markdown
}

func newPagerModel(general *general) pagerModel {
	// Init viewport
	vp := viewport.Model{}
	vp.YPosition = 0
	vp.HighPerformanceRendering = config.HighPerformancePager

	// Text input for notes/memos
	ti := textinput.NewModel()
	ti.Prompt = te.String(" > ").
		Foreground(common.Color(darkGray)).
		Background(common.YellowGreen.Color()).
		String()
	ti.TextColor = darkGray
	ti.BackgroundColor = common.YellowGreen.String()
	ti.CursorColor = common.Fuschia.String()
	ti.CharLimit = noteCharacterLimit
	ti.Focus()

	// Text input for search
	sp := spinner.NewModel()
	sp.ForegroundColor = statusBarNoteFg.String()
	sp.BackgroundColor = statusBarBg.String()
	sp.HideFor = time.Millisecond * 50
	sp.MinimumLifetime = time.Millisecond * 180

	return pagerModel{
		general:   general,
		state:     pagerStateBrowse,
		textInput: ti,
		viewport:  vp,
		spinner:   sp,
	}
}

func (m *pagerModel) setSize(w, h int) {
	m.viewport.Width = w
	m.viewport.Height = h - statusBarHeight
	m.textInput.Width = w -
		ansi.PrintableRuneWidth(noteHeading) -
		ansi.PrintableRuneWidth(m.textInput.Prompt) - 1

	if m.showHelp {
		if pagerHelpHeight == 0 {
			pagerHelpHeight = strings.Count(m.helpView(), "\n")
		}
		m.viewport.Height -= (statusBarHeight + pagerHelpHeight)
	}
}

func (m *pagerModel) setContent(s string) {
	m.viewport.SetContent(s)
}

func (m *pagerModel) toggleHelp() {
	m.showHelp = !m.showHelp
	m.setSize(m.general.width, m.general.height)
	if m.viewport.PastBottom() {
		m.viewport.GotoBottom()
	}
}

// Perform stuff that needs to happen after a successful markdown stash. Note
// that the the returned command should be sent back the through the pager
// update function.
func (m *pagerModel) showStatusMessage(statusMessage string) tea.Cmd {
	// Show a success message to the user
	m.state = pagerStateStatusMessage
	m.statusMessage = statusMessage
	if m.statusMessageTimer != nil {
		m.statusMessageTimer.Stop()
	}
	m.statusMessageTimer = time.NewTimer(statusMessageTimeout)

	return waitForStatusMessageTimeout(pagerContext, m.statusMessageTimer)
}

func (m *pagerModel) unload() {
	if m.showHelp {
		m.toggleHelp()
	}
	if m.statusMessageTimer != nil {
		m.statusMessageTimer.Stop()
	}
	m.state = pagerStateBrowse
	m.viewport.SetContent("")
	m.viewport.YOffset = 0
	m.textInput.Reset()
}

func (m pagerModel) Update(msg tea.Msg) (pagerModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case pagerStateSetNote:
			switch msg.String() {
			case "esc":
				m.state = pagerStateBrowse
				return m, nil
			case "enter":
				var cmd tea.Cmd
				if m.textInput.Value() != m.currentDocument.Note { // don't update if the note didn't change
					m.currentDocument.Note = m.textInput.Value() // update optimistically
					cmd = saveDocumentNote(m.general.cc, m.currentDocument.ID, m.currentDocument.Note)
				}
				m.state = pagerStateBrowse
				m.textInput.Reset()
				return m, cmd
			}
		default:
			switch msg.String() {
			case "q", "esc":
				if m.state != pagerStateBrowse {
					m.state = pagerStateBrowse
					return m, nil
				}
			case "home", "g":
				m.viewport.GotoTop()
				if m.viewport.HighPerformanceRendering {
					cmds = append(cmds, viewport.Sync(m.viewport))
				}
			case "end", "G":
				m.viewport.GotoBottom()
				if m.viewport.HighPerformanceRendering {
					cmds = append(cmds, viewport.Sync(m.viewport))
				}
			case "m":
				isStashed := m.currentDocument.markdownType == stashedMarkdown ||
					m.currentDocument.markdownType == convertedMarkdown

				// Users can only set the note on user-stashed markdown
				if !isStashed {
					break
				}

				m.state = pagerStateSetNote

				// Stop the timer for hiding a status message since changing
				// the state above will have cleared it.
				if m.statusMessageTimer != nil {
					m.statusMessageTimer.Stop()
				}

				// Pre-populate note with existing value
				if m.textInput.Value() == "" {
					m.textInput.SetValue(m.currentDocument.Note)
					m.textInput.CursorEnd()
				}

				return m, textinput.Blink
			case "s":
				if m.general.authStatus != authOK {
					break
				}

				// Stash a local document
				if m.state != pagerStateStashing && m.currentDocument.markdownType == localMarkdown {
					m.state = pagerStateStashing
					m.spinner.Start()
					cmds = append(
						cmds,
						stashDocument(m.general.cc, m.currentDocument),
						spinner.Tick,
					)
				}
			case "?":
				m.toggleHelp()
				if m.viewport.HighPerformanceRendering {
					cmds = append(cmds, viewport.Sync(m.viewport))
				}
			}
		}

	case spinner.TickMsg:
		if m.state == pagerStateStashing || m.spinner.Visible() {
			newSpinnerModel, cmd := m.spinner.Update(msg)
			m.spinner = newSpinnerModel
			cmds = append(cmds, cmd)
		} else if m.state == pagerStateStashSuccess && !m.spinner.Visible() {
			m.state = pagerStateBrowse
			m.currentDocument = *m.stashedDocument
			m.stashedDocument = nil
			cmd := m.showStatusMessage("Stashed!")
			cmds = append(cmds, cmd)
		}

	// Glow has rendered the content
	case contentRenderedMsg:
		m.setContent(string(msg))
		if m.viewport.HighPerformanceRendering {
			cmds = append(cmds, viewport.Sync(m.viewport))
		}

	// We've reveived terminal dimensions, either for the first time or
	// after a resize
	case tea.WindowSizeMsg:
		return m, renderWithGlamour(m, m.currentDocument.Body)

	case stashSuccessMsg:
		// Stashing was successful. Convert the loaded document to a stashed
		// one and show a status message. Note that we're also handling this
		// message in the main update function where we're adding this stashed
		// item to the stash listing.
		m.state = pagerStateStashSuccess
		if !m.spinner.Visible() {
			m.state = pagerStateBrowse
			m.currentDocument = markdown(msg)
			cmd := m.showStatusMessage("Stashed!")
			cmds = append(cmds, cmd)
		} else {
			md := markdown(msg)
			m.stashedDocument = &md
		}

	case stashErrMsg:
		// TODO

	case statusMessageTimeoutMsg:
		// Hide the status message bar
		m.state = pagerStateBrowse
	}

	switch m.state {
	case pagerStateSetNote:
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	default:
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m pagerModel) View() string {
	var b strings.Builder
	fmt.Fprint(&b, m.viewport.View()+"\n")

	// Footer
	switch m.state {
	case pagerStateSetNote:
		m.setNoteView(&b)
	default:
		m.statusBarView(&b)
	}

	if m.showHelp {
		fmt.Fprint(&b, m.helpView())
	}

	return b.String()
}

func (m pagerModel) statusBarView(b *strings.Builder) {
	const (
		minPercent               float64 = 0.0
		maxPercent               float64 = 1.0
		percentToStringMagnitude float64 = 100.0
	)
	var (
		isStashed         bool = m.currentDocument.markdownType == stashedMarkdown || m.currentDocument.markdownType == convertedMarkdown
		showStatusMessage bool = m.state == pagerStateStatusMessage
	)

	// Logo
	logo := glowLogoView(" Glow ")

	// Scroll percent
	percent := math.Max(minPercent, math.Min(maxPercent, m.viewport.ScrollPercent()))
	scrollPercent := fmt.Sprintf(" %3.f%% ", percent*percentToStringMagnitude)
	if showStatusMessage {
		scrollPercent = statusBarMessageScrollPosStyle(scrollPercent)
	} else {
		scrollPercent = statusBarScrollPosStyle(scrollPercent)
	}

	// "Help" note
	var helpNote string
	if showStatusMessage {
		helpNote = statusBarMessageHelpStyle(" ? Help ")
	} else {
		helpNote = statusBarHelpStyle(" ? Help ")
	}

	// Status indicator; spinner or stash dot
	var statusIndicator string
	if m.state == pagerStateStashing || m.state == pagerStateStashSuccess {
		if m.spinner.Visible() {
			statusIndicator = statusBarNoteStyle(" ") + m.spinner.View()
		}
	} else if isStashed && showStatusMessage {
		statusIndicator = statusBarMessageStashIconStyle(" " + pagerStashIcon)
	} else if isStashed {
		statusIndicator = statusBarStashDotStyle(" " + pagerStashIcon)
	}

	// Note
	var note string
	if showStatusMessage {
		note = m.statusMessage
	} else {
		note = m.currentDocument.Note
		if len(note) == 0 {
			note = "(No memo)"
		}
	}
	note = truncate(" "+note+" ", max(0,
		m.general.width-
			ansi.PrintableRuneWidth(logo)-
			ansi.PrintableRuneWidth(statusIndicator)-
			ansi.PrintableRuneWidth(scrollPercent)-
			ansi.PrintableRuneWidth(helpNote),
	))
	if showStatusMessage {
		note = statusBarMessageStyle(note)
	} else {
		note = statusBarNoteStyle(note)
	}

	// Empty space
	padding := max(0,
		m.general.width-
			ansi.PrintableRuneWidth(logo)-
			ansi.PrintableRuneWidth(statusIndicator)-
			ansi.PrintableRuneWidth(note)-
			ansi.PrintableRuneWidth(scrollPercent)-
			ansi.PrintableRuneWidth(helpNote),
	)
	emptySpace := strings.Repeat(" ", padding)
	if showStatusMessage {
		emptySpace = statusBarMessageStyle(emptySpace)
	} else {
		emptySpace = statusBarNoteStyle(emptySpace)
	}

	fmt.Fprintf(b, "%s%s%s%s%s%s",
		logo,
		statusIndicator,
		note,
		emptySpace,
		scrollPercent,
		helpNote,
	)
}

func (m pagerModel) setNoteView(b *strings.Builder) {
	fmt.Fprint(b, noteHeading)
	fmt.Fprint(b, m.textInput.View())
}

func (m pagerModel) helpView() (s string) {
	memoOrStash := "m       set memo"
	if m.general.authStatus == authOK && m.currentDocument.markdownType == localMarkdown {
		memoOrStash = "s       stash this document"
	}

	col1 := []string{
		"g/home  go to top",
		"G/end   go to bottom",
		"",
		memoOrStash,
		"esc     back to files",
		"q       quit",
	}

	if m.currentDocument.markdownType == newsMarkdown {
		deleteFromStringSlice(col1, 3)
	}

	s += "\n"
	s += "k/↑      up                  " + col1[0] + "\n"
	s += "j/↓      down                " + col1[1] + "\n"
	s += "b/pgup   page up             " + col1[2] + "\n"
	s += "f/pgdn   page down           " + col1[3] + "\n"
	s += "u        ½ page up           " + col1[4] + "\n"
	s += "d        ½ page down         "

	if len(col1) > 5 {
		s += col1[5]
	}

	s = indent(s, 2)

	// Fill up empty cells with spaces for background coloring
	if m.general.width > 0 {
		lines := strings.Split(s, "\n")
		for i := 0; i < len(lines); i++ {
			l := runewidth.StringWidth(lines[i])
			n := max(m.general.width-l, 0)
			lines[i] += strings.Repeat(" ", n)
		}

		s = strings.Join(lines, "\n")
	}

	return helpViewStyle(s)
}

// COMMANDS

func renderWithGlamour(m pagerModel, md string) tea.Cmd {
	return func() tea.Msg {
		s, err := glamourRender(m, md)
		if err != nil {
			if debug {
				log.Println("error rendering with Glamour:", err)
			}
			return errMsg{err}
		}
		return contentRenderedMsg(s)
	}
}

// This is where the magic happens.
func glamourRender(m pagerModel, markdown string) (string, error) {
	if !config.GlamourEnabled {
		return markdown, nil
	}

	// initialize glamour
	var gs glamour.TermRendererOption
	if m.general.cfg.GlamourStyle == "auto" {
		gs = glamour.WithAutoStyle()
	} else {
		gs = glamour.WithStylePath(m.general.cfg.GlamourStyle)
	}

	width := max(0, min(int(m.general.cfg.GlamourMaxWidth), m.viewport.Width))
	r, err := glamour.NewTermRenderer(
		gs,
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}

	out, err := r.Render(markdown)
	if err != nil {
		return "", err
	}

	// trim lines
	lines := strings.Split(out, "\n")

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

// ETC

// Note: this runs in linear time; O(n).
func deleteFromStringSlice(a []string, i int) []string {
	copy(a[i:], a[i+1:])
	a[len(a)-1] = ""
	return a[:len(a)-1]
}
