package ui

import (
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	pagerStateJumpToLine
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

	// Search
	searchInput   textinput.Model
	searchQuery   string // active search term (persists after input is confirmed)
	searchMatches []int  // line numbers with matches (0-indexed into raw Body lines)
	searchIndex   int    // current match index (-1 = none)

	// Jump to line
	lineInput textinput.Model

	watcher *fsnotify.Watcher
}

func newPagerModel(common *commonModel) pagerModel {
	// Init viewport
	vp := viewport.New(0, 0)
	vp.YPosition = 0
	vp.HighPerformanceRendering = config.HighPerformancePager

	si := textinput.New()
	si.Prompt = "/"
	si.PromptStyle = lipgloss.NewStyle().Foreground(yellowGreen)
	si.Focus()

	li := textinput.New()
	li.Prompt = ":"
	li.PromptStyle = lipgloss.NewStyle().Foreground(yellowGreen)
	li.Focus()

	m := pagerModel{
		common:      common,
		state:       pagerStateBrowse,
		viewport:    vp,
		searchInput: si,
		searchIndex: -1,
		lineInput:   li,
	}
	m.initWatcher()
	return m
}

func (m *pagerModel) setSize(w, h int) {
	m.viewport.Width = w
	m.viewport.Height = h - statusBarHeight

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

// inInputMode returns true when the pager is in a state that consumes
// arbitrary key input (search prompt, jump prompt) or has active search
// results that esc should clear before unloading the document.
func (m pagerModel) inInputMode() bool {
	return m.state == pagerStateSearch ||
		m.state == pagerStateJumpToLine ||
		m.searchQuery != ""
}

func (m *pagerModel) clearSearch() {
	m.searchQuery = ""
	m.searchMatches = nil
	m.searchIndex = -1
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
	m.clearSearch()
	m.viewport.SetContent("")
	m.viewport.YOffset = 0
	m.unwatchFile()
}

// findMatches finds all line numbers in content that contain the query
// (case-insensitive). Returns 0-indexed line numbers.
func findMatches(content string, query string) []int {
	lines := strings.Split(content, "\n")
	lowerQuery := strings.ToLower(query)
	var matches []int
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), lowerQuery) {
			matches = append(matches, i)
		}
	}
	return matches
}

func (m pagerModel) update(msg tea.Msg) (pagerModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case pagerStateSearch:
			cmds = append(cmds, m.handleSearchInput(msg))
			return m, tea.Batch(cmds...)

		case pagerStateJumpToLine:
			cmds = append(cmds, m.handleJumpInput(msg))
			return m, tea.Batch(cmds...)

		case pagerStateStatusMessage:
			// Any key returns to browse
			m.state = pagerStateBrowse
			return m, nil

		case pagerStateBrowse:
			cmds = append(cmds, m.handleBrowseKeys(msg))
		}

	// Glow has rendered the content
	case contentRenderedMsg:
		log.Info("content rendered", "state", m.state)

		m.setContent(string(msg))
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
		// Only transition to browse if we're actually showing a status message.
		// Ignore if in search/jump input mode.
		if m.state == pagerStateStatusMessage {
			m.state = pagerStateBrowse
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *pagerModel) handleBrowseKeys(msg tea.KeyMsg) tea.Cmd {
	var cmds []tea.Cmd

	switch msg.String() {
	case keyEsc:
		// If search results are active, clear them
		if m.searchQuery != "" {
			m.clearSearch()
			return nil
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

	case "right":
		m.viewport.ViewDown()
		if m.viewport.HighPerformanceRendering {
			cmds = append(cmds, viewport.Sync(m.viewport))
		}

	case "left":
		m.viewport.ViewUp()
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
		return openEditor(m.currentDocument.localPath, lineno)

	case "c":
		// Copy using OSC 52
		termenv.Copy(m.currentDocument.Body)
		// Copy using native system clipboard
		_ = clipboard.WriteAll(m.currentDocument.Body)
		cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{"Copied contents", false}))

	case "r":
		return loadLocalMarkdown(&m.currentDocument)

	case "?":
		m.toggleHelp()
		if m.viewport.HighPerformanceRendering {
			cmds = append(cmds, viewport.Sync(m.viewport))
		}

	case "/":
		m.state = pagerStateSearch
		m.searchInput.SetValue("")
		m.searchInput.Focus()
		return textinput.Blink

	case ":":
		m.state = pagerStateJumpToLine
		m.lineInput.SetValue("")
		m.lineInput.Focus()
		return textinput.Blink

	case "n":
		if m.searchQuery != "" && len(m.searchMatches) > 0 {
			m.searchIndex++
			if m.searchIndex >= len(m.searchMatches) {
				m.searchIndex = 0
				cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{"search wrapped", false}))
			}
			m.viewport.SetYOffset(m.searchMatches[m.searchIndex])
			if m.viewport.HighPerformanceRendering {
				cmds = append(cmds, viewport.Sync(m.viewport))
			}
		}

	case "N":
		if m.searchQuery != "" && len(m.searchMatches) > 0 {
			m.searchIndex--
			if m.searchIndex < 0 {
				m.searchIndex = len(m.searchMatches) - 1
				cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{"search wrapped", false}))
			}
			m.viewport.SetYOffset(m.searchMatches[m.searchIndex])
			if m.viewport.HighPerformanceRendering {
				cmds = append(cmds, viewport.Sync(m.viewport))
			}
		}
	}

	return tea.Batch(cmds...)
}

func (m *pagerModel) handleSearchInput(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case keyEnter:
		query := m.searchInput.Value()
		if query == "" {
			m.state = pagerStateBrowse
			return m.showStatusMessage(pagerStatusMessage{"no pattern", false})
		}
		m.searchQuery = query
		m.searchMatches = findMatches(m.currentDocument.Body, query)
		if len(m.searchMatches) == 0 {
			m.searchQuery = ""
			m.state = pagerStateBrowse
			return m.showStatusMessage(pagerStatusMessage{"no matches", false})
		}
		m.searchIndex = 0
		m.viewport.SetYOffset(m.searchMatches[0])
		m.state = pagerStateBrowse
		if m.viewport.HighPerformanceRendering {
			return viewport.Sync(m.viewport)
		}
		return nil

	case keyEsc:
		m.clearSearch()
		m.state = pagerStateBrowse
		return nil
	}

	// Delegate to the text input
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return cmd
}

func (m *pagerModel) handleJumpInput(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case keyEnter:
		input := m.lineInput.Value()
		m.state = pagerStateBrowse
		if input == "" {
			return nil
		}

		// Check for percentage jump (e.g., "50%")
		if strings.HasSuffix(input, "%") {
			pct, err := strconv.Atoi(strings.TrimSuffix(input, "%"))
			if err != nil {
				return m.showStatusMessage(pagerStatusMessage{"invalid number", false})
			}
			pct = max(0, min(100, pct))
			if pct == 0 {
				m.viewport.GotoTop()
			} else if pct == 100 {
				m.viewport.GotoBottom()
			} else {
				totalLines := m.viewport.TotalLineCount()
				target := int(math.Round(float64(totalLines) * float64(pct) / 100))
				m.viewport.SetYOffset(target)
			}
		} else {
			n, err := strconv.Atoi(input)
			if err != nil {
				return m.showStatusMessage(pagerStatusMessage{"invalid line number", false})
			}
			n = max(1, min(n, m.viewport.TotalLineCount()))
			m.viewport.SetYOffset(n - 1) // convert 1-indexed to 0-indexed
		}

		if m.viewport.HighPerformanceRendering {
			return viewport.Sync(m.viewport)
		}
		return nil

	case keyEsc:
		m.state = pagerStateBrowse
		return nil
	}

	// Delegate to the text input
	var cmd tea.Cmd
	m.lineInput, cmd = m.lineInput.Update(msg)
	return cmd
}

func (m pagerModel) View() string {
	var b strings.Builder
	fmt.Fprint(&b, m.viewport.View()+"\n")

	// Footer
	m.statusBarView(&b)

	if m.showHelp {
		fmt.Fprint(&b, "\n"+m.helpView())
	}

	return b.String()
}

func (m pagerModel) statusBarView(b *strings.Builder) {
	// When in search or jump input mode, replace the entire status bar
	// with the input prompt (like less/vim).
	if m.state == pagerStateSearch {
		fmt.Fprint(b, m.searchInput.View())
		// Pad to full width
		inputWidth := ansi.PrintableRuneWidth(m.searchInput.View())
		if pad := m.common.width - inputWidth; pad > 0 {
			fmt.Fprint(b, statusBarNoteStyle(strings.Repeat(" ", pad)))
		}
		return
	}
	if m.state == pagerStateJumpToLine {
		fmt.Fprint(b, m.lineInput.View())
		inputWidth := ansi.PrintableRuneWidth(m.lineInput.View())
		if pad := m.common.width - inputWidth; pad > 0 {
			fmt.Fprint(b, statusBarNoteStyle(strings.Repeat(" ", pad)))
		}
		return
	}

	const (
		minPercent               float64 = 0.0
		maxPercent               float64 = 1.0
		percentToStringMagnitude float64 = 100.0
	)

	showStatusMessage := m.state == pagerStateStatusMessage

	// Logo
	logo := glowLogoView()

	// Page indicator
	var pageIndicator string
	viewHeight := max(1, m.viewport.Height)
	currentPage := m.viewport.YOffset/viewHeight + 1
	totalPages := (m.viewport.TotalLineCount() + viewHeight - 1) / viewHeight
	currentPage = min(currentPage, totalPages)
	if totalPages > 1 {
		pageIndicator = fmt.Sprintf(" pg %d/%d ", currentPage, totalPages)
	}
	if showStatusMessage {
		pageIndicator = statusBarMessageScrollPosStyle(pageIndicator)
	} else {
		pageIndicator = statusBarScrollPosStyle(pageIndicator)
	}

	// Match counter (when search results are active)
	var matchCounter string
	if m.searchQuery != "" && len(m.searchMatches) > 0 {
		matchCounter = fmt.Sprintf(" %d/%d ", m.searchIndex+1, len(m.searchMatches))
	}
	if showStatusMessage {
		matchCounter = statusBarMessageScrollPosStyle(matchCounter)
	} else {
		matchCounter = statusBarScrollPosStyle(matchCounter)
	}

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
	} else {
		note = m.currentDocument.Note
	}
	note = truncate.StringWithTail(" "+note+" ", uint(max(0, //nolint:gosec
		m.common.width-
			ansi.PrintableRuneWidth(logo)-
			ansi.PrintableRuneWidth(matchCounter)-
			ansi.PrintableRuneWidth(pageIndicator)-
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
			ansi.PrintableRuneWidth(matchCounter)-
			ansi.PrintableRuneWidth(pageIndicator)-
			ansi.PrintableRuneWidth(scrollPercent)-
			ansi.PrintableRuneWidth(helpNote),
	)
	emptySpace := strings.Repeat(" ", padding)
	if showStatusMessage {
		emptySpace = statusBarMessageStyle(emptySpace)
	} else {
		emptySpace = statusBarNoteStyle(emptySpace)
	}

	fmt.Fprintf(b, "%s%s%s%s%s%s%s",
		logo,
		note,
		emptySpace,
		matchCounter,
		pageIndicator,
		scrollPercent,
		helpNote,
	)
}

func (m pagerModel) helpView() (s string) {
	col1 := []string{
		"g/home  go to top",
		"G/end   go to bottom",
		"/       search",
		"n/N     next/prev match",
		":       jump to line/pct",
		"c       copy contents",
		"e       edit this document",
		"r       reload this document",
		"esc     back to files",
		"q       quit",
	}

	s += "\n"
	s += "k/↑      up                  " + col1[0] + "\n"
	s += "j/↓      down                " + col1[1] + "\n"
	s += "b/pgup   page up             " + col1[2] + "\n"
	s += "f/pgdn   page down           " + col1[3] + "\n"
	s += "←/→      page back/fwd       " + col1[4] + "\n"
	s += "u        ½ page up           " + col1[5] + "\n"
	s += "d        ½ page down         " + col1[6] + "\n"
	s += "                             " + col1[7] + "\n"
	s += "                             " + col1[8]

	if len(col1) > 9 {
		s += "\n                             " + col1[9]
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
