package ui

import (
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glow/v2/utils"
	"github.com/charmbracelet/lipgloss"
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

var (
	pagerHelpHeight int

	mintGreen = lipgloss.AdaptiveColor{Light: "#89F0CB", Dark: "#89F0CB"}
	darkGreen = lipgloss.AdaptiveColor{Light: "#1C8760", Dark: "#1C8760"}

	lineNumberFg = lipgloss.AdaptiveColor{Light: "#656565", Dark: "#7D7D7D"}

	statusBarNoteFg = lipgloss.AdaptiveColor{Light: "#656565", Dark: "#7D7D7D"}
	statusBarBg     = lipgloss.AdaptiveColor{Light: "#E6E6E6", Dark: "#242424"}

	statusBarScrollPosStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#949494", Dark: "#5A5A5A"}).
				Background(statusBarBg).
				Render

	statusBarNoteStyle = lipgloss.NewStyle().
				Foreground(statusBarNoteFg).
				Background(statusBarBg).
				Render

	statusBarHelpStyle = lipgloss.NewStyle().
				Foreground(statusBarNoteFg).
				Background(lipgloss.AdaptiveColor{Light: "#DCDCDC", Dark: "#323232"}).
				Render

	statusBarMessageStyle = lipgloss.NewStyle().
				Foreground(mintGreen).
				Background(darkGreen).
				Render

	statusBarMessageScrollPosStyle = lipgloss.NewStyle().
					Foreground(mintGreen).
					Background(darkGreen).
					Render

	statusBarMessageHelpStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#B6FFE4")).
					Background(green).
					Render

	helpViewStyle = lipgloss.NewStyle().
			Foreground(statusBarNoteFg).
			Background(lipgloss.AdaptiveColor{Light: "#f2f2f2", Dark: "#1B1B1B"}).
			Render

	lineNumberStyle = lipgloss.NewStyle().
			Foreground(lineNumberFg).
			Render

	// Search-related styles
	searchHighlightStyle = lipgloss.NewStyle().
				Background(lipgloss.AdaptiveColor{Light: "#FFFF00", Dark: "#FFFF00"}).
				Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#000000"}).
				Bold(true)

	searchCurrentHighlightStyle = lipgloss.NewStyle().
					Background(lipgloss.AdaptiveColor{Light: "#FF6600", Dark: "#FF6600"}).
					Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#000000"}).
					Bold(true)

	searchInputPromptStyle = lipgloss.NewStyle().
				Foreground(yellowGreen).
				MarginRight(1)

	searchInputCursorStyle = lipgloss.NewStyle().
				Foreground(fuchsia)
)

type (
	contentRenderedMsg string
	reloadMsg          struct{}
)

type pagerState int

const (
	pagerStateBrowse pagerState = iota
	pagerStateStatusMessage
	pagerStateSearch
)

// searchMatch represents a match position in the content
type searchMatch struct {
	lineIndex int // line number (0-based)
	startCol  int // start column in the line
	endCol    int // end column in the line
}

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

	// Search-related fields
	searchInput       textinput.Model
	searchQuery       string
	searchMatches     []searchMatch
	currentMatchIndex int
	renderedContent   string   // full rendered content (for searching)
	renderedLines     []string // cached split lines (avoid repeated splits)
	plainLines        []string // cached ANSI-stripped lines (avoid repeated stripAnsi)
	searchActive      bool     // whether search results are being displayed
	searchError       string   // error message for invalid pattern
}

func newPagerModel(common *commonModel) pagerModel {
	// Init viewport
	vp := viewport.New(0, 0)
	vp.YPosition = 0
	vp.HighPerformanceRendering = config.HighPerformancePager

	// Init search input
	si := textinput.New()
	si.Prompt = "/"
	si.PromptStyle = searchInputPromptStyle
	si.Cursor.Style = searchInputCursorStyle
	si.Placeholder = "search..."

	m := pagerModel{
		common:            common,
		state:             pagerStateBrowse,
		viewport:          vp,
		searchInput:       si,
		currentMatchIndex: -1,
	}
	m.initWatcher()
	return m
}

func (m *pagerModel) setSize(w, h int) {
	m.viewport.Width = w
	m.viewport.Height = h - statusBarHeight

	// Set search input width
	m.searchInput.Width = w - 4 // Account for prompt and padding

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

// setRenderedContent updates the rendered content and caches processed lines.
func (m *pagerModel) setRenderedContent(content string) {
	m.renderedContent = content
	m.renderedLines = strings.Split(content, "\n")
	// Pre-strip ANSI codes for faster searching
	m.plainLines = make([]string, len(m.renderedLines))
	for i, line := range m.renderedLines {
		m.plainLines[i] = stripAnsi(line)
	}
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
	if m.showHelp {
		m.toggleHelp()
	}
	if m.statusMessageTimer != nil {
		m.statusMessageTimer.Stop()
	}
	m.state = pagerStateBrowse
	m.viewport.SetContent("")
	m.viewport.YOffset = 0
	m.unwatchFile()
	m.clearSearch()
}

func (m *pagerModel) clearSearch() {
	m.searchQuery = ""
	m.searchMatches = nil
	m.currentMatchIndex = -1
	m.searchActive = false
	m.searchError = ""
	m.searchInput.Reset()
	// Restore original content if we have it
	if m.renderedContent != "" {
		m.viewport.SetContent(m.renderedContent)
	}
}

func (m pagerModel) update(msg tea.Msg) (pagerModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// Handle search mode input
	if m.state == pagerStateSearch {
		return m.handleSearchInput(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			if m.state != pagerStateBrowse {
				m.state = pagerStateBrowse
				return m, nil
			}
		case keyEsc:
			if m.searchActive {
				// Clear search and restore original content
				m.clearSearch()
				if m.viewport.HighPerformanceRendering {
					cmds = append(cmds, viewport.Sync(m.viewport))
				}
				return m, tea.Batch(cmds...)
			}
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

		case "d":
			m.viewport.HalfViewDown()
			if m.viewport.HighPerformanceRendering {
				cmds = append(cmds, viewport.Sync(m.viewport))
			}

		case "u":
			m.viewport.HalfViewUp()
			if m.viewport.HighPerformanceRendering {
				cmds = append(cmds, viewport.Sync(m.viewport))
			}

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
			m.clearSearch()
			return m, loadLocalMarkdown(&m.currentDocument)

		case "?":
			m.toggleHelp()
			if m.viewport.HighPerformanceRendering {
				cmds = append(cmds, viewport.Sync(m.viewport))
			}

		case "/":
			// Enter search mode
			m.state = pagerStateSearch
			m.searchInput.Focus()
			return m, textinput.Blink

		case "n":
			// Next search match
			if m.searchActive && len(m.searchMatches) > 0 {
				m.currentMatchIndex = (m.currentMatchIndex + 1) % len(m.searchMatches)
				m.jumpToCurrentMatch()
				m.updateSearchHighlighting()
				if m.viewport.HighPerformanceRendering {
					cmds = append(cmds, viewport.Sync(m.viewport))
				}
			}

		case "N":
			// Previous search match
			if m.searchActive && len(m.searchMatches) > 0 {
				m.currentMatchIndex--
				if m.currentMatchIndex < 0 {
					m.currentMatchIndex = len(m.searchMatches) - 1
				}
				m.jumpToCurrentMatch()
				m.updateSearchHighlighting()
				if m.viewport.HighPerformanceRendering {
					cmds = append(cmds, viewport.Sync(m.viewport))
				}
			}
		}

	// Glow has rendered the content
	case contentRenderedMsg:
		log.Info("content rendered", "state", m.state)

		// Store the full rendered content and cache split lines for searching
		m.setRenderedContent(string(msg))
		m.setContent(m.renderedContent)

		// Re-perform search if there's an active search query
		// This handles terminal resizes which re-render the content
		if m.searchActive && m.searchQuery != "" {
			m.performSearch()
			if len(m.searchMatches) > 0 {
				// Clamp currentMatchIndex to valid range
				if m.currentMatchIndex >= len(m.searchMatches) {
					m.currentMatchIndex = len(m.searchMatches) - 1
				}
				if m.currentMatchIndex < 0 {
					m.currentMatchIndex = 0
				}
				m.jumpToCurrentMatch()
				m.updateSearchHighlighting()
			}
		}

		if m.viewport.HighPerformanceRendering {
			cmds = append(cmds, viewport.Sync(m.viewport))
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

	// Show search input if in search mode
	if m.state == pagerStateSearch {
		m.searchBarView(&b)
	} else {
		// Footer
		m.statusBarView(&b)
	}

	if m.showHelp {
		fmt.Fprint(&b, "\n"+m.helpView())
	}

	return b.String()
}

func (m pagerModel) searchBarView(b *strings.Builder) {
	// Search input
	searchInput := m.searchInput.View()

	// Pad with spaces to fill the width
	padding := max(0, m.common.width-ansi.PrintableRuneWidth(searchInput))
	paddedInput := searchInput + strings.Repeat(" ", padding)

	fmt.Fprint(b, statusBarNoteStyle(paddedInput))
}

func (m pagerModel) statusBarView(b *strings.Builder) {
	const (
		minPercent               float64 = 0.0
		maxPercent               float64 = 1.0
		percentToStringMagnitude float64 = 100.0
	)

	showStatusMessage := m.state == pagerStateStatusMessage

	// Logo
	logo := glowLogoView()

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

	// Note
	var note string
	if showStatusMessage {
		note = m.statusMessage
	} else if m.searchActive && m.searchError != "" {
		note = fmt.Sprintf("[%s] %s", m.searchError, m.searchQuery)
	} else if m.searchActive && len(m.searchMatches) > 0 {
		note = fmt.Sprintf("[%d/%d] %s", m.currentMatchIndex+1, len(m.searchMatches), m.searchQuery)
	} else if m.searchActive && len(m.searchMatches) == 0 {
		note = fmt.Sprintf("[no matches] %s", m.searchQuery)
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
		note = statusBarMessageStyle(note)
	} else {
		note = statusBarNoteStyle(note)
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
		emptySpace = statusBarMessageStyle(emptySpace)
	} else {
		emptySpace = statusBarNoteStyle(emptySpace)
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
		"/       search",
		"n/N     next/prev match",
		"esc     clear search / back",
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
	s += "                             " + col1[7] + "\n"
	s += "                             " + col1[8]

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

// SEARCH FUNCTIONS

// ansiPattern matches ANSI escape sequences
var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x07]*\x07|\x1b[PX^_][^\x1b]*\x1b\\|\x1b.`)

// stripAnsi removes ANSI escape codes from a string
func stripAnsi(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

// highlightRenderedLine inserts search highlights into an ANSI-formatted line
// at the correct plain-text positions, preserving existing ANSI formatting.
func highlightRenderedLine(rendered string, matches []searchMatch, currentMatchIndex, lineIndex int) string {
	if len(matches) == 0 {
		return rendered
	}

	// Filter matches for this line
	var lineMatches []searchMatch
	for _, m := range matches {
		if m.lineIndex == lineIndex {
			lineMatches = append(lineMatches, m)
		}
	}
	if len(lineMatches) == 0 {
		return rendered
	}

	var result strings.Builder
	plainPos := 0  // Current position in plain text (what user sees)
	i := 0         // Current position in rendered string
	matchIdx := 0  // Current match we're processing
	inHighlight := false

	for i < len(rendered) {
		// Check for ANSI escape sequence
		if rendered[i] == '\x1b' {
			// Find the end of the ANSI sequence
			loc := ansiPattern.FindStringIndex(rendered[i:])
			if loc != nil && loc[0] == 0 {
				// Copy the ANSI sequence as-is
				result.WriteString(rendered[i : i+loc[1]])
				i += loc[1]
				continue
			}
		}

		// Check if we need to start a highlight
		if matchIdx < len(lineMatches) && plainPos == lineMatches[matchIdx].startCol && !inHighlight {
			// Find if this match is the current one (for different highlight style)
			isCurrent := false
			for globalIdx, m := range matches {
				if m.lineIndex == lineIndex && m.startCol == lineMatches[matchIdx].startCol {
					isCurrent = (globalIdx == currentMatchIndex)
					break
				}
			}
			if isCurrent {
				result.WriteString("\x1b[48;2;255;102;0m\x1b[38;2;0;0;0m\x1b[1m") // Orange bg, black fg, bold
			} else {
				result.WriteString("\x1b[48;2;255;255;0m\x1b[38;2;0;0;0m\x1b[1m") // Yellow bg, black fg, bold
			}
			inHighlight = true
		}

		// Check if we need to end a highlight
		if matchIdx < len(lineMatches) && plainPos == lineMatches[matchIdx].endCol && inHighlight {
			result.WriteString("\x1b[0m") // Reset
			inHighlight = false
			matchIdx++
		}

		// Copy the current character
		result.WriteByte(rendered[i])
		plainPos++
		i++
	}

	// Close any open highlight at end of line
	if inHighlight {
		result.WriteString("\x1b[0m")
	}

	return result.String()
}

func (m pagerModel) handleSearchInput(msg tea.Msg) (pagerModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case keyEsc:
			// Cancel search and restore original content
			m.state = pagerStateBrowse
			m.searchInput.Reset()
			m.clearSearch()
			if m.viewport.HighPerformanceRendering {
				cmds = append(cmds, viewport.Sync(m.viewport))
			}
			return m, tea.Batch(cmds...)

		case keyEnter:
			// Confirm search and exit search mode
			m.state = pagerStateBrowse
			query := m.searchInput.Value()
			if query == "" {
				m.clearSearch()
				return m, nil
			}
			// Search is already done incrementally, just confirm it
			if m.viewport.HighPerformanceRendering {
				cmds = append(cmds, viewport.Sync(m.viewport))
			}
			return m, tea.Batch(cmds...)
		}
	}

	// Update the search input
	var cmd tea.Cmd
	previousValue := m.searchInput.Value()
	m.searchInput, cmd = m.searchInput.Update(msg)
	cmds = append(cmds, cmd)

	// Incremental search: if the input value changed, perform search
	newValue := m.searchInput.Value()
	if newValue != previousValue {
		m.searchQuery = newValue
		if newValue == "" {
			// Clear search if input is empty
			m.searchMatches = nil
			m.searchActive = false
			m.currentMatchIndex = -1
			// Restore original content
			if m.renderedContent != "" {
				m.viewport.SetContent(m.renderedContent)
			}
		} else {
			// Perform incremental search
			m.performSearch()
			if len(m.searchMatches) > 0 {
				m.currentMatchIndex = 0
				m.jumpToCurrentMatch()
				m.updateSearchHighlighting()
			} else {
				// No matches - restore original content but keep search active
				if m.renderedContent != "" {
					m.viewport.SetContent(m.renderedContent)
				}
			}
		}
		if m.viewport.HighPerformanceRendering {
			cmds = append(cmds, viewport.Sync(m.viewport))
		}
	}

	return m, tea.Batch(cmds...)
}

// performSearch finds all matches in the content using simple string matching.
func (m *pagerModel) performSearch() {
	m.searchMatches = nil
	m.searchError = ""

	if m.searchQuery == "" || len(m.plainLines) == 0 {
		m.searchActive = false
		return
	}

	m.searchActive = true

	// Smart case: case-insensitive unless query has uppercase letters
	caseSensitive := false
	for _, r := range m.searchQuery {
		if unicode.IsUpper(r) {
			caseSensitive = true
			break
		}
	}

	query := m.searchQuery
	queryLen := len(query)
	if !caseSensitive {
		query = strings.ToLower(query)
	}

	// Find all matches using simple string search
	for lineIdx, plainLine := range m.plainLines {
		searchLine := plainLine
		if !caseSensitive {
			searchLine = strings.ToLower(plainLine)
		}

		// Find all occurrences in this line
		offset := 0
		for {
			idx := strings.Index(searchLine[offset:], query)
			if idx == -1 {
				break
			}
			startCol := offset + idx
			m.searchMatches = append(m.searchMatches, searchMatch{
				lineIndex: lineIdx,
				startCol:  startCol,
				endCol:    startCol + queryLen,
			})
			offset = startCol + queryLen
		}
	}
}

func (m *pagerModel) jumpToCurrentMatch() {
	if m.currentMatchIndex < 0 || m.currentMatchIndex >= len(m.searchMatches) {
		return
	}

	match := m.searchMatches[m.currentMatchIndex]

	// Calculate the target line, centering it in the viewport
	targetLine := match.lineIndex - m.viewport.Height/2
	if targetLine < 0 {
		targetLine = 0
	}

	m.viewport.YOffset = targetLine
}

// updateSearchHighlighting applies highlighting to matched text while preserving ANSI formatting.
func (m *pagerModel) updateSearchHighlighting() {
	if !m.searchActive || len(m.searchMatches) == 0 || len(m.renderedLines) == 0 {
		return
	}

	// Track which lines have matches
	linesWithMatches := make(map[int]bool)
	for _, match := range m.searchMatches {
		linesWithMatches[match.lineIndex] = true
	}

	// Copy lines and apply highlighting only to lines with matches
	lines := make([]string, len(m.renderedLines))
	copy(lines, m.renderedLines)

	for lineIdx := range linesWithMatches {
		if lineIdx < len(lines) {
			lines[lineIdx] = highlightRenderedLine(m.renderedLines[lineIdx], m.searchMatches, m.currentMatchIndex, lineIdx)
		}
	}

	m.viewport.SetContent(strings.Join(lines, "\n"))
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
	trunc := lipgloss.NewStyle().MaxWidth(m.viewport.Width - lineNumberWidth).Render

	if !config.GlamourEnabled {
		return markdown, nil
	}

	isCode := !utils.IsMarkdownFile(m.currentDocument.Note)
	width := max(0, min(int(m.common.cfg.GlamourMaxWidth), m.viewport.Width)) //nolint:gosec
	if isCode {
		width = 0
	}

	options := []glamour.TermRendererOption{
		utils.GlamourStyle(m.common.cfg.GlamourStyle, isCode),
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
	}

	// trim lines
	lines := strings.Split(out, "\n")

	var content strings.Builder
	for i, s := range lines {
		if isCode || m.common.cfg.ShowLineNumbers {
			content.WriteString(lineNumberStyle(fmt.Sprintf("%"+fmt.Sprint(lineNumberWidth)+"d", i+1)))
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

			return reloadMsg{}
		case _, ok := <-m.watcher.Errors:
			if !ok {
				continue
			}
		}
	}
}

func (m *pagerModel) unwatchFile() {
	dir := m.localDir()

	_ = m.watcher.Remove(dir)
}

func (m *pagerModel) localDir() string {
	return filepath.Dir(m.currentDocument.localPath)
}
