package ui

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/glow/v3/utils"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/log"
	"github.com/fsnotify/fsnotify"
	runewidth "github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/ansi"
	"github.com/muesli/reflow/truncate"
	"github.com/muesli/termenv"
)

const (
	statusBarHeight = 1
	lineNumberWidth = 4
)

var pagerHelpHeight int

type (
	contentRenderedMsg string
	reloadMsg          struct{}
)

type pagerState int

const (
	pagerStateBrowse pagerState = iota
	pagerStateStatusMessage
	pagerStateSearch // user is actively typing a search query
)

type pagerModel struct {
	common   *commonModel
	viewport viewport.Model
	state    pagerState
	showHelp bool

	statusMessage      string
	statusMessageTimer *time.Timer

	// Current document being rendered, sans-glamour rendering. We cache
	// it here so we can re-render it on resize.
	currentDocument markdown

	watcher *fsnotify.Watcher

	// Search state
	searchInput   textinput.Model // bottom search bar input
	searchQuery   string          // last applied query
	searchMatches []int           // line indices (0-based) containing a match
	searchCursor  int             // index into searchMatches for current highlighted match
	rawContent    string          // glamour-rendered content without search highlights
}

func newPagerModel(common *commonModel) pagerModel {
	// Init viewport
	vp := viewport.New()

	// Init search input
	si := textinput.New()
	si.Prompt = "/"
	si.SetVirtualCursor(true)
	tsi := si.Styles()
	tsi.Focused.Prompt = common.styles.stashInputPromptStyle
	tsi.Blurred.Prompt = common.styles.stashInputPromptStyle
	tsi.Cursor.Color = common.styles.fuchsia
	si.SetStyles(tsi)

	m := pagerModel{
		common:      common,
		state:       pagerStateBrowse,
		viewport:    vp,
		searchInput: si,
	}
	m.initWatcher()
	return m
}

func (m *pagerModel) setSize(w, h int) {
	m.viewport.SetWidth(w)
	vh := h - statusBarHeight
	if m.state == pagerStateSearch {
		vh-- // reserve one line for the search bar
	}
	m.viewport.SetHeight(vh)

	if m.showHelp {
		if pagerHelpHeight == 0 {
			pagerHelpHeight = strings.Count(m.helpView(), "\n")
		}
		m.viewport.SetHeight(m.viewport.Height() - (statusBarHeight + pagerHelpHeight))
	}
}

func (m *pagerModel) setContent(s string) {
	m.rawContent = s
	m.viewport.SetContent(s)
}

func (m *pagerModel) toggleHelp() {
	m.showHelp = !m.showHelp
	m.setSize(m.common.width, m.common.height)
	if m.viewport.PastBottom() {
		m.viewport.GotoBottom()
	}
}

type pagerStatusMessage struct {
	message string
	isError bool
}

// Perform stuff that needs to happen after a successful markdown stash. Note
// that the returned command should be sent back the through the pager
// update function.
func (m *pagerModel) showStatusMessage(msg pagerStatusMessage) tea.Cmd {
	// Show a success message to the user
	m.state = pagerStateStatusMessage
	m.statusMessage = msg.message
	if m.statusMessageTimer != nil {
		m.statusMessageTimer.Stop()
	}
	m.statusMessageTimer = time.NewTimer(statusMessageTimeout)

	return waitForStatusMessageTimeout(pagerContext, m.statusMessageTimer)
}

func (m *pagerModel) unload() {
	log.Debug("unload")
	if m.showHelp {
		m.toggleHelp()
	}
	if m.statusMessageTimer != nil {
		m.statusMessageTimer.Stop()
	}
	m.state = pagerStateBrowse
	m.viewport.SetContent("")
	m.viewport.SetYOffset(0)
	m.unwatchFile()
	// Reset search state
	m.searchQuery = ""
	m.searchMatches = nil
	m.searchCursor = 0
	m.rawContent = ""
	m.searchInput.Reset()
}

func (m pagerModel) update(msg tea.Msg) (pagerModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// --- Search mode: intercept all keys ---
		if m.state == pagerStateSearch {
			switch msg.String() {
			case keyEsc:
				// Exit search: restore un-highlighted content
				m.state = pagerStateBrowse
				m.searchQuery = ""
				m.searchMatches = nil
				m.searchCursor = 0
				m.searchInput.Reset()
				m.setSize(m.common.width, m.common.height)
				m.viewport.SetContent(m.rawContent)
				return m, nil
			case keyEnter:
				// Apply search
				q := m.searchInput.Value()
				if q != "" {
					m.searchQuery = q
					m.applySearch()
				}
				m.state = pagerStateBrowse
				m.setSize(m.common.width, m.common.height)
				return m, nil
			default:
				// Forward remaining keys to textinput
				var inputCmd tea.Cmd
				m.searchInput, inputCmd = m.searchInput.Update(msg)
				cmds = append(cmds, inputCmd)
				return m, tea.Batch(cmds...)
			}
		}

		// --- Normal browse mode ---
		switch msg.String() {
		case "q", keyEsc:
			if m.state != pagerStateBrowse {
				m.state = pagerStateBrowse
				return m, nil
			}
		case "home", "g":
			m.viewport.GotoTop()
		case "end", "G":
			m.viewport.GotoBottom()

		case "d":
			m.viewport.HalfPageDown()

		case "u":
			m.viewport.HalfPageUp()

		case "e":
			lineno := int(math.RoundToEven(float64(m.viewport.TotalLineCount()) * m.viewport.ScrollPercent()))
			if m.viewport.AtTop() {
				lineno = 0
			}
			log.Info(
				"opening editor",
				"file", m.currentDocument.localPath,
				"line", fmt.Sprintf("%d/%d", lineno, m.viewport.TotalLineCount()),
			)
			return m, openEditor(m.currentDocument.localPath, lineno)

		case "c":
			// Copy using OSC 52
			termenv.Copy(m.currentDocument.Body)
			// Copy using native system clipboard
			_ = clipboard.WriteAll(m.currentDocument.Body)
			cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{"Copied contents", false}))

		case "r":
			return m, loadLocalMarkdown(&m.currentDocument)

		case "/":
			// Enter search mode
			m.state = pagerStateSearch
			m.searchInput.Reset()
			m.searchInput.Focus()
			m.setSize(m.common.width, m.common.height)
			return m, textinput.Blink

		case "n":
			// Jump to next search match
			if len(m.searchMatches) > 0 {
				m.searchCursor = (m.searchCursor + 1) % len(m.searchMatches)
				m.viewport.SetYOffset(m.searchMatches[m.searchCursor])
			}

		case "N":
			// Jump to previous search match
			if len(m.searchMatches) > 0 {
				m.searchCursor = (m.searchCursor - 1 + len(m.searchMatches)) % len(m.searchMatches)
				m.viewport.SetYOffset(m.searchMatches[m.searchCursor])
			}

		case "?":
			m.toggleHelp()
		}

	// Glow has rendered the content
	case contentRenderedMsg:
		log.Info("content rendered", "state", m.state)

		m.setContent(string(msg))
		// Re-apply search highlights if a query is active (e.g. after resize)
		if m.searchQuery != "" {
			m.applySearch()
		}
		cmds = append(cmds, m.watchFile)

	// The file was changed on disk and we're reloading it
	case reloadMsg:
		return m, loadLocalMarkdown(&m.currentDocument)

	// We've finished editing the document, potentially making changes. Let's
	// retrieve the latest version of the document so that we display
	// up-to-date contents.
	case editorFinishedMsg:
		return m, loadLocalMarkdown(&m.currentDocument)

	// We've received terminal dimensions, either for the first time or
	// after a resize
	case tea.WindowSizeMsg:
		return m, renderWithGlamour(m, m.currentDocument.Body)

	case statusMessageTimeoutMsg:
		m.state = pagerStateBrowse
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m pagerModel) View() string {
	var b strings.Builder
	fmt.Fprint(&b, m.viewport.View()+"\n")

	// Search bar (shown between viewport and status bar during search mode)
	if m.state == pagerStateSearch {
		fmt.Fprint(&b, m.searchBarView()+"\n")
	}

	// Footer
	m.statusBarView(&b)

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

	showStatusMessage := m.state == pagerStateStatusMessage
	styles := m.common.styles

	// Logo
	logo := glowLogoView(m.common.styles)

	// Scroll percent
	percent := math.Max(minPercent, math.Min(maxPercent, m.viewport.ScrollPercent()))
	scrollPercent := fmt.Sprintf(" %3.f%% ", percent*percentToStringMagnitude)
	if showStatusMessage {
		scrollPercent = styles.statusBarMessageScrollPosStyle(scrollPercent)
	} else {
		scrollPercent = styles.statusBarScrollPosStyle(scrollPercent)
	}

	// "Help" note
	var helpNote string
	if showStatusMessage {
		helpNote = styles.statusBarMessageHelpStyle(" ? Help ")
	} else {
		helpNote = styles.statusBarHelpStyle(" ? Help ")
	}

	// Note
	var note string
	if showStatusMessage {
		note = m.statusMessage
	} else {
		note = m.currentDocument.Note
	}
	note = truncate.StringWithTail(" "+note+" ", uint(max(0, //nolint:gosec
		m.common.width-
			ansi.PrintableRuneWidth(logo)-
			ansi.PrintableRuneWidth(scrollPercent)-
			ansi.PrintableRuneWidth(helpNote),
	)), ellipsis)
	if showStatusMessage {
		note = styles.statusBarMessageStyle(note)
	} else {
		note = styles.statusBarNoteStyle(note)
	}

	// Empty space
	padding := max(0,
		m.common.width-
			ansi.PrintableRuneWidth(logo)-
			ansi.PrintableRuneWidth(note)-
			ansi.PrintableRuneWidth(scrollPercent)-
			ansi.PrintableRuneWidth(helpNote),
	)
	emptySpace := strings.Repeat(" ", padding)
	if showStatusMessage {
		emptySpace = styles.statusBarMessageStyle(emptySpace)
	} else {
		emptySpace = styles.statusBarNoteStyle(emptySpace)
	}

	fmt.Fprintf(b, "%s%s%s%s%s",
		logo,
		note,
		emptySpace,
		scrollPercent,
		helpNote,
	)
}

func (m pagerModel) helpView() (s string) {
	col1 := []string{
		"g/home  go to top",
		"G/end   go to bottom",
		"c       copy contents",
		"e       edit this document",
		"r       reload this document",
		"/       search in document",
		"n/N     next/prev match",
		"esc     back to files",
		"q       quit",
	}

	s += "\n"
	s += "k/↑      up                  " + col1[0] + "\n"
	s += "j/↓      down                " + col1[1] + "\n"
	s += "b/pgup   page up             " + col1[2] + "\n"
	s += "f/pgdn   page down           " + col1[3] + "\n"
	s += "u        ½ page up           " + col1[4] + "\n"
	s += "d        ½ page down         " + col1[5] + "\n"
	s += "                             " + col1[6] + "\n"
	s += "                             " + col1[7]

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

	return m.common.styles.helpViewStyle(s)
}

// COMMANDS

func renderWithGlamour(m pagerModel, md string) tea.Cmd {
	return func() tea.Msg {
		s, err := glamourRender(m, md)
		if err != nil {
			log.Error("error rendering with Glamour", "error", err)
			return errMsg{err}
		}
		return contentRenderedMsg(s)
	}
}

// This is where the magic happens.
func glamourRender(m pagerModel, markdown string) (string, error) {
	trunc := lipgloss.NewStyle().MaxWidth(m.viewport.Width() - lineNumberWidth).Render

	if !m.common.cfg.GlamourEnabled {
		return markdown, nil
	}

	isCode := !utils.IsMarkdownFile(m.currentDocument.Note)
	width := max(0, min(int(m.common.cfg.GlamourMaxWidth), m.viewport.Width())) //nolint:gosec
	if isCode {
		width = 0
	} else if m.common.cfg.ShowLineNumbers && width > lineNumberWidth {
		width -= lineNumberWidth
	}

	options := []glamour.TermRendererOption{
		utils.GlamourStyle(m.common.cfg.GlamourStyle, isCode, width),
		glamour.WithWordWrap(width),
	}

	if m.common.cfg.PreserveNewLines {
		options = append(options, glamour.WithPreservedNewLines())
	}
	r, err := glamour.NewTermRenderer(options...)
	if err != nil {
		return "", fmt.Errorf("error creating glamour renderer: %w", err)
	}

	if isCode {
		markdown = utils.WrapCodeBlock(markdown, filepath.Ext(m.currentDocument.Note))
	}

	out, err := r.Render(markdown)
	if err != nil {
		return "", fmt.Errorf("error rendering markdown: %w", err)
	}

	if isCode {
		out = strings.TrimSpace(out)
	} else {
		out = utils.ApplyTextSizing(out)
		out = utils.ApplyCodeBlockBackground(out)
	}

	// trim lines
	lines := strings.Split(out, "\n")

	var content strings.Builder
	for i, s := range lines {
		if isCode || m.common.cfg.ShowLineNumbers {
			content.WriteString(m.common.styles.lineNumberStyle(fmt.Sprintf("%"+fmt.Sprint(lineNumberWidth)+"d", i+1)))
			content.WriteString(trunc(s))
		} else {
			content.WriteString(s)
		}

		// don't add an artificial newline after the last split
		if i+1 < len(lines) {
			content.WriteRune('\n')
		}
	}

	return content.String(), nil
}

func (m *pagerModel) initWatcher() {
	var err error
	m.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Error("error creating fsnotify watcher", "error", err)
	}
}

func (m *pagerModel) watchFile() tea.Msg {
	dir := m.localDir()

	if err := m.watcher.Add(dir); err != nil {
		log.Error("error adding dir to fsnotify watcher", "error", err)
		return nil
	}

	log.Info("fsnotify watching dir", "dir", dir)

	for {
		select {
		case event, ok := <-m.watcher.Events:
			if !ok || event.Name != m.currentDocument.localPath {
				continue
			}

			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
				continue
			}

			log.Debug("fsnotify event", "file", event.Name, "event", event.Op)
			return reloadMsg{}
		case err, ok := <-m.watcher.Errors:
			if !ok {
				continue
			}
			log.Debug("fsnotify error", "dir", dir, "error", err)
		}
	}
}

func (m *pagerModel) unwatchFile() {
	dir := m.localDir()

	err := m.watcher.Remove(dir)
	if err == nil {
		log.Debug("fsnotify dir unwatched", "dir", dir)
	} else {
		log.Error("fsnotify fail to unwatch dir", "dir", dir, "error", err)
	}
}

func (m *pagerModel) localDir() string {
	return filepath.Dir(m.currentDocument.localPath)
}

// applySearch scans rawContent for lines matching searchQuery (case-insensitive),
// injects lipgloss highlight spans, updates viewport content, and records match line
// indices so that n/N can jump between them.
func (m *pagerModel) applySearch() {
	if m.searchQuery == "" || m.rawContent == "" {
		return
	}

	hlStyle := m.common.styles.searchHighlightStyle
	q := strings.ToLower(m.searchQuery)

	lines := strings.Split(m.rawContent, "\n")
	m.searchMatches = m.searchMatches[:0] // reset, reuse slice

	var highlighted strings.Builder
	for i, line := range lines {
		// We search in the raw line (which may contain ANSI codes from glamour).
		// User-typed queries never contain escape codes, so a case-insensitive
		// strings.Index on the lowercased line is sufficient for matching.
		lineLower := strings.ToLower(line)

		if idx := strings.Index(lineLower, q); idx >= 0 {
			// Record this line as a match (0-based)
			m.searchMatches = append(m.searchMatches, i)

			// Inject highlight around the found substring in the raw line.
			// The byte offset is valid because q is ASCII-range text.
			before := line[:idx]
			match := line[idx : idx+len(m.searchQuery)]
			after := line[idx+len(m.searchQuery):]
			highlighted.WriteString(before + hlStyle.Render(match) + after)
		} else {
			highlighted.WriteString(line)
		}

		if i+1 < len(lines) {
			highlighted.WriteRune('\n')
		}
	}

	m.viewport.SetContent(highlighted.String())

	// Jump to first match
	if len(m.searchMatches) > 0 {
		m.searchCursor = 0
		m.viewport.SetYOffset(m.searchMatches[0])
	}
}

// searchBarView renders the one-line search input bar shown above the status bar.
func (m pagerModel) searchBarView() string {
	styles := m.common.styles
	w := m.common.width

	prompt := styles.searchBarPromptStyle("/ Search: ")
	input := m.searchInput.View()

	// ansi.PrintableRuneWidth strips ANSI codes internally when measuring width,
	// so we can measure prompt and input directly.
	promptW := ansi.PrintableRuneWidth(prompt)
	inputW := ansi.PrintableRuneWidth(input)
	pad := max(0, w-promptW-inputW)
	padding := strings.Repeat(" ", pad)

	return styles.searchBarStyle(prompt + input + padding)
}
