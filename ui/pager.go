package ui

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"path"
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

const (
	maxDocumentWidth = 120
	statusBarHeight  = 1
	gray             = "#333333"
	yellowGreen      = "#ECFD65"
	fuschia          = "#EE6FF8"
	mintGreen        = "#89F0CB"
	darkGreen        = "#1C8760"
	noteHeadingText  = " Set Memo "
	notePromptText   = " > "
)

var (
	pagerHelpHeight = strings.Count(pagerHelpView(pagerModel{}, 0), "\n")

	noteHeading = te.String(noteHeadingText).
			Foreground(common.Cream.Color()).
			Background(common.Green.Color()).
			String()

	statusBarBg     = common.NewColorPair("#242424", "#E6E6E6")
	statusBarNoteFg = common.NewColorPair("#7D7D7D", "#656565")

	// Styling functions.
	statusBarScrollPosStyle        = newStyle(common.NewColorPair("#5A5A5A", "#949494").String(), statusBarBg.String())
	statusBarNoteStyle             = newStyle(statusBarNoteFg.String(), statusBarBg.String())
	statusBarHelpStyle             = newStyle(statusBarNoteFg.String(), common.NewColorPair("#323232", "#DCDCDC").String())
	statusBarStashDotStyle         = newStyle(common.Green.String(), statusBarBg.String())
	statusBarMessageHeaderStyle    = newStyle(common.Cream.String(), common.Green.String())
	statusBarMessageBodyStyle      = newStyle(mintGreen, darkGreen)
	statusBarMessageStashDotStyle  = newStyle(mintGreen, darkGreen)
	statusBarMessageScrollPosStyle = newStyle(mintGreen, darkGreen)
	statusBarMessageHelpStyle      = newStyle("#B6FFE4", common.Green.String())

	helpViewStyle = newStyle(statusBarNoteFg.String(), common.NewColorPair("#1B1B1B", "#f2f2f2").String())
)

// Create a new termenv styling function.
func newStyle(fg, bg string) func(string) string {
	return te.Style{}.
		Foreground(te.ColorProfile().Color(fg)).
		Background(te.ColorProfile().Color(bg)).
		Styled
}

// MSG

type contentRenderedMsg string
type noteSavedMsg *charm.Markdown
type stashSuccessMsg markdown
type stashErrMsg struct {
	err error
}

func (s stashErrMsg) Error() string {
	return s.err.Error()
}

// MODEL

type pagerState int

const (
	pagerStateBrowse pagerState = iota
	pagerStateSetNote
	pagerStateStashing
	pagerStateStatusMessage
)

type pagerModel struct {
	cc           *charm.Client
	viewport     viewport.Model
	state        pagerState
	glamourStyle string
	width        int
	height       int
	textInput    textinput.Model
	showHelp     bool
	spinner      spinner.Model

	statusMessageHeader string
	statusMessageBody   string
	statusMessageTimer  *time.Timer

	// Current document being rendered, sans-glamour rendering. We cache
	// it here so we can re-render it on resize.
	currentDocument markdown
}

func newPagerModel(glamourStyle string) pagerModel {

	// Init viewport
	vp := viewport.Model{}
	vp.YPosition = 0
	vp.HighPerformanceRendering = config.HighPerformancePager

	// Init text input UI for notes/memos
	ti := textinput.NewModel()
	ti.Prompt = te.String(notePromptText).
		Foreground(te.ColorProfile().Color(gray)).
		Background(te.ColorProfile().Color(yellowGreen)).
		String()
	ti.TextColor = gray
	ti.BackgroundColor = yellowGreen
	ti.CursorColor = fuschia
	ti.CharLimit = noteCharacterLimit
	ti.Focus()

	sp := spinner.NewModel()
	sp.ForegroundColor = statusBarNoteFg.String()
	sp.BackgroundColor = statusBarBg.String()

	return pagerModel{
		state:        pagerStateBrowse,
		glamourStyle: glamourStyle,
		textInput:    ti,
		viewport:     vp,
		spinner:      sp,
	}
}

func (m *pagerModel) setSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h - statusBarHeight
	m.textInput.Width = w - len(noteHeadingText) - len(notePromptText) - 1

	if m.showHelp {
		m.viewport.Height -= (statusBarHeight + pagerHelpHeight)
	}
}

func (m *pagerModel) setContent(s string) {
	m.viewport.SetContent(s)
}

func (m *pagerModel) toggleHelp() {
	m.showHelp = !m.showHelp
	m.setSize(m.width, m.height)
	if m.viewport.PastBottom() {
		m.viewport.GotoBottom()
	}
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

// UPDATE

func pagerUpdate(msg tea.Msg, m pagerModel) (pagerModel, tea.Cmd) {
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
					cmd = saveDocumentNote(m.cc, m.currentDocument.ID, m.currentDocument.Note)
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

				return m, textinput.Blink(m.textInput)
			case "s":
				// Stash a local document
				if m.state != pagerStateStashing && m.currentDocument.markdownType == localMarkdown {
					m.state = pagerStateStashing
					cmds = append(
						cmds,
						stashDocument(m.cc, m.currentDocument),
						spinner.Tick(m.spinner),
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
		if m.state == pagerStateStashing {
			newSpinnerModel, cmd := spinner.Update(msg, m.spinner)
			m.spinner = newSpinnerModel
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
		// one. Note that we're also handling this message in the main update
		// function where we're adding this stashed item to the stash listing.

		m.state = pagerStateBrowse

		// Replace the current document in the state so its metadata becomes
		// that of a stashed document, but don't re-render since the body is
		// identical to what's already rendered.
		m.currentDocument = markdown(msg)

		// Show a success message to the user.
		m.state = pagerStateStatusMessage
		m.statusMessageHeader = "Stashed!"
		m.statusMessageBody = ""
		if m.statusMessageTimer != nil {
			m.statusMessageTimer.Stop()
		}
		m.statusMessageTimer = time.NewTimer(statusMessageTimeout)
		cmds = append(cmds, waitForStatusMessageTimeout(pagerContext, m.statusMessageTimer))

	case stashErrMsg:
		// TODO

	case statusMessageTimeoutMsg:
		// Hide the status message bar
		m.state = pagerStateBrowse
	}

	switch m.state {
	case pagerStateSetNote:
		m.textInput, cmd = textinput.Update(msg, m.textInput)
		cmds = append(cmds, cmd)
	default:
		m.viewport, cmd = viewport.Update(msg, m.viewport)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// VIEW

func pagerView(m pagerModel) string {
	var b strings.Builder

	fmt.Fprint(&b, viewport.View(m.viewport)+"\n")

	// Footer
	switch m.state {
	case pagerStateSetNote:
		pagerSetNoteView(&b, m)
	default:
		pagerStatusBarView(&b, m)
	}

	if m.showHelp {
		fmt.Fprintf(&b, pagerHelpView(m, m.width))
	}

	return b.String()
}

func pagerStatusBarView(b *strings.Builder, m pagerModel) {
	var (
		isStashed         bool = m.currentDocument.markdownType == stashedMarkdown || m.currentDocument.markdownType == convertedMarkdown
		showStatusMessage bool = m.state == pagerStateStatusMessage
	)

	// Logo
	logo := glowLogoView(" Glow ")

	// Scroll percent
	percent := math.Max(0.0, math.Min(1.0, m.viewport.ScrollPercent()))
	scrollPercent := fmt.Sprintf(" %3.f%% ", percent*100)
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
	if m.state == pagerStateStashing {
		statusIndicator = statusBarNoteStyle(" ") + spinner.View(m.spinner)
	} else if isStashed && showStatusMessage {
		statusIndicator = statusBarMessageStashDotStyle(" •")
	} else if isStashed {
		statusIndicator = statusBarStashDotStyle(" •")
	}

	// Note
	var note string
	if showStatusMessage {
		note = "Stashed!"
	} else {
		note = m.currentDocument.Note
		if len(note) == 0 {
			note = "(No memo)"
		}
	}
	note = truncate(" "+note+" ", max(0,
		m.width-
			ansi.PrintableRuneWidth(logo)-
			ansi.PrintableRuneWidth(statusIndicator)-
			ansi.PrintableRuneWidth(scrollPercent)-
			ansi.PrintableRuneWidth(helpNote),
	))
	if showStatusMessage {
		note = statusBarMessageBodyStyle(note)
	} else {
		note = statusBarNoteStyle(note)
	}

	// Empty space
	padding := max(0,
		m.width-
			ansi.PrintableRuneWidth(logo)-
			ansi.PrintableRuneWidth(statusIndicator)-
			ansi.PrintableRuneWidth(note)-
			ansi.PrintableRuneWidth(scrollPercent)-
			ansi.PrintableRuneWidth(helpNote),
	)
	emptySpace := strings.Repeat(" ", padding)
	if showStatusMessage {
		emptySpace = statusBarMessageBodyStyle(emptySpace)
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

func pagerSetNoteView(b *strings.Builder, m pagerModel) {
	fmt.Fprint(b, noteHeading)
	fmt.Fprint(b, textinput.View(m.textInput))
}

func pagerHelpView(m pagerModel, width int) (s string) {
	col1 := [...]string{
		"m       set memo",
		"esc     back to files",
		"q       quit",
	}
	if m.currentDocument.markdownType != stashedMarkdown {
		col1[0] = "s       stash this document"
	}

	s += "\n"
	s += "k/↑      up                  " + col1[0] + "\n"
	s += "j/↓      down                " + col1[1] + "\n"
	s += "j/↓      down                " + col1[2] + "\n"
	s += "b/pgup   page up\n"
	s += "f/pgdn   page down\n"
	s += "u        ½ page up\n"
	s += "d        ½ page down"

	s = indent(s, 2)

	// Fill up empty cells with spaces for background coloring
	if width > 0 {
		lines := strings.Split(s, "\n")
		for i := 0; i < len(lines); i++ {
			l := runewidth.StringWidth(lines[i])
			n := max(width-l, 0)
			lines[i] += strings.Repeat(" ", n)
		}

		s = strings.Join(lines, "\n")
	}

	return helpViewStyle(s)
}

// CMD

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

func saveDocumentNote(cc *charm.Client, id int, note string) tea.Cmd {
	if cc == nil {
		return func() tea.Msg {
			return errMsg{errors.New("can't set note; no charm client")}
		}
	}
	return func() tea.Msg {
		if err := cc.SetMarkdownNote(id, note); err != nil {
			if debug {
				log.Println("error saving note:", err)
			}
			return errMsg{err}
		}
		return noteSavedMsg(&charm.Markdown{ID: id, Note: note})
	}
}

func stashDocument(cc *charm.Client, md markdown) tea.Cmd {
	return func() tea.Msg {
		if cc == nil {
			return func() tea.Msg {
				err := errors.New("can't stash; no charm client")
				if debug {
					log.Println("error stash document:", err)
				}
				return stashErrMsg{err}
			}
		}

		// Is the document missing a body? If so, it likely means it needs to
		// be loaded. If the document body is really empty then we'll still
		// stash it.
		if len(md.Body) == 0 {
			data, err := ioutil.ReadFile(md.localPath)
			if err != nil {
				if debug {
					log.Println("error loading doucument body for stashing:", err)
				}
				return stashErrMsg{err}
			}
			md.Body = string(data)
		}

		// Turn local markdown into a newly stashed (converted) markdown
		md.markdownType = convertedMarkdown
		md.CreatedAt = time.Now()

		// Set the note as the filename without the extension
		p := md.localPath
		md.Note = strings.Replace(path.Base(p), path.Ext(p), "", 1)

		newMd, err := cc.StashMarkdown(md.Note, md.Body)
		if err != nil {
			if debug {
				log.Println("error stashing document:", err)
			}
			return stashErrMsg{err}
		}

		// We really just need to know the ID so we can operate on this newly
		// stashed markdown.
		md.ID = newMd.ID
		return stashSuccessMsg(md)
	}
}

// This is where the magic happens.
func glamourRender(m pagerModel, markdown string) (string, error) {

	if !config.GlamourEnabled {
		return markdown, nil
	}

	// initialize glamour
	var gs glamour.TermRendererOption
	if m.glamourStyle == "auto" {
		gs = glamour.WithAutoStyle()
	} else {
		gs = glamour.WithStylePath(m.glamourStyle)
	}

	width := max(0, min(maxDocumentWidth, m.viewport.Width))
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
