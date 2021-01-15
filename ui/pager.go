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
	lib "github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/glamour"
	runewidth "github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/ansi"
	"github.com/muesli/reflow/truncate"
	te "github.com/muesli/termenv"
)

const statusBarHeight = 1

var (
	pagerHelpHeight int

	mintGreen = lib.NewColorPair("#89F0CB", "#89F0CB")
	darkGreen = lib.NewColorPair("#1C8760", "#1C8760")

	noteHeading = te.String(" Set Memo ").
			Foreground(lib.Cream.Color()).
			Background(lib.Green.Color()).
			String()

	statusBarNoteFg = lib.NewColorPair("#7D7D7D", "#656565")
	statusBarBg     = lib.NewColorPair("#242424", "#E6E6E6")

	// Styling funcs.
	statusBarScrollPosStyle        = newStyle(lib.NewColorPair("#5A5A5A", "#949494"), statusBarBg, false)
	statusBarNoteStyle             = newStyle(statusBarNoteFg, statusBarBg, false)
	statusBarHelpStyle             = newStyle(statusBarNoteFg, lib.NewColorPair("#323232", "#DCDCDC"), false)
	statusBarStashDotStyle         = newStyle(lib.Green, statusBarBg, false)
	statusBarMessageStyle          = newStyle(mintGreen, darkGreen, false)
	statusBarMessageStashIconStyle = newStyle(mintGreen, darkGreen, false)
	statusBarMessageScrollPosStyle = newStyle(mintGreen, darkGreen, false)
	statusBarMessageHelpStyle      = newStyle(lib.NewColorPair("#B6FFE4", "#B6FFE4"), lib.Green, false)
	helpViewStyle                  = newStyle(statusBarNoteFg, lib.NewColorPair("#1B1B1B", "#f2f2f2"), false)
)

type contentRenderedMsg string
type noteSavedMsg *charm.Markdown

type pagerState int

const (
	pagerStateBrowse pagerState = iota
	pagerStateSetNote
	pagerStateStashing
	pagerStateStashSuccess
	pagerStateStatusMessage
)

type pagerModel struct {
	common    *commonModel
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

func newPagerModel(common *commonModel) pagerModel {
	// Init viewport
	vp := viewport.Model{}
	vp.YPosition = 0
	vp.HighPerformanceRendering = config.HighPerformancePager

	// Text input for notes/memos
	ti := textinput.NewModel()
	ti.Prompt = te.String(" > ").
		Foreground(lib.Color(darkGray)).
		Background(lib.YellowGreen.Color()).
		String()
	ti.TextColor = darkGray
	ti.BackgroundColor = lib.YellowGreen.String()
	ti.CursorColor = lib.Fuschia.String()
	ti.CharLimit = noteCharacterLimit
	ti.Focus()

	// Text input for search
	sp := spinner.NewModel()
	sp.ForegroundColor = statusBarNoteFg.String()
	sp.BackgroundColor = statusBarBg.String()
	sp.HideFor = time.Millisecond * 50
	sp.MinimumLifetime = time.Millisecond * 180

	return pagerModel{
		common:    common,
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
	m.setSize(m.common.width, m.common.height)
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

func (m pagerModel) update(msg tea.Msg) (pagerModel, tea.Cmd) {
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
					cmd = saveDocumentNote(m.common.cc, m.currentDocument.ID, m.currentDocument.Note)
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
				isStashed := m.currentDocument.docType == StashedDoc ||
					m.currentDocument.docType == ConvertedDoc

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
				if m.common.authStatus != authOK {
					break
				}

				md := m.currentDocument

				_, alreadyStashed := m.common.filesStashed[md.stashID]
				if alreadyStashed {
					cmds = append(cmds, m.showStatusMessage("Already stashed"))
					break
				}

				// Stash a local document
				if m.state != pagerStateStashing && stashableDocTypes.Contains(md.docType) {
					m.state = pagerStateStashing
					m.spinner.Start()
					cmds = append(
						cmds,
						stashDocument(m.common.cc, md),
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
			// If we're still stashing, or if the spinner still needs to
			// finish, spin it along.
			newSpinnerModel, cmd := m.spinner.Update(msg)
			m.spinner = newSpinnerModel
			cmds = append(cmds, cmd)
		} else if m.state == pagerStateStashSuccess && !m.spinner.Visible() {
			// If the spinner's finished and we haven't told the user the
			// stash was successful, do that.
			m.state = pagerStateBrowse
			m.currentDocument = *m.stashedDocument
			m.stashedDocument = nil
			cmds = append(cmds, m.showStatusMessage("Stashed!"))
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
			// The spinner has finished spinning, so tell the user the stash
			// was successful.
			m.state = pagerStateBrowse
			m.currentDocument = markdown(msg)
			cmds = append(cmds, m.showStatusMessage("Stashed!"))
		} else {
			// The spinner is still spinning, so just take note of the newly
			// stashed document for now.
			md := markdown(msg)
			m.stashedDocument = &md
		}

	case stashFailMsg:
		delete(m.common.filesStashed, msg.markdown.stashID)

	case statusMessageTimeoutMsg:
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
		fmt.Fprint(&b, "\n"+m.helpView())
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
		isStashed         bool = m.currentDocument.docType == StashedDoc || m.currentDocument.docType == ConvertedDoc
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
	note = truncate.StringWithTail(" "+note+" ", uint(max(0,
		m.common.width-
			ansi.PrintableRuneWidth(logo)-
			ansi.PrintableRuneWidth(statusIndicator)-
			ansi.PrintableRuneWidth(scrollPercent)-
			ansi.PrintableRuneWidth(helpNote),
	)), ellipsis)
	if showStatusMessage {
		note = statusBarMessageStyle(note)
	} else {
		note = statusBarNoteStyle(note)
	}

	// Empty space
	padding := max(0,
		m.common.width-
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
	if m.common.authStatus == authOK && m.currentDocument.docType == LocalDoc {
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

	if m.currentDocument.docType == NewsDoc {
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
	if m.common.width > 0 {
		lines := strings.Split(s, "\n")
		for i := 0; i < len(lines); i++ {
			l := runewidth.StringWidth(lines[i])
			n := max(m.common.width-l, 0)
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
	if m.common.cfg.GlamourStyle == "auto" {
		gs = glamour.WithAutoStyle()
	} else {
		gs = glamour.WithStylePath(m.common.cfg.GlamourStyle)
	}

	width := max(0, min(int(m.common.cfg.GlamourMaxWidth), m.viewport.Width))
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
