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

	// maxSearchQueryDisplayWidth caps how much of the search query is shown
	// in the status bar, so a long query can't crowd out the match count.
	maxSearchQueryDisplayWidth = 30
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
	pagerStateSearch
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

	// Search
	searchInput     textinput.Model
	searchQuery     string
	searchMatches   []searchMatch
	searchIndex     int // index into searchMatches currently selected; -1 if none
	searching       bool
	renderedContent string // last glamour-rendered content, before any search highlighting is baked in
}

func newPagerModel(common *commonModel) pagerModel {
	// Init viewport
	vp := viewport.New()

	// Init search input
	si := textinput.New()
	si.Prompt = "/"
	si.SetVirtualCursor(true)
	tsi := si.Styles()
	tsi.Focused.Prompt = common.styles.inputPromptStyle
	tsi.Blurred.Prompt = common.styles.inputPromptStyle
	tsi.Cursor.Color = common.styles.fuchsia
	si.SetStyles(tsi)

	m := pagerModel{
		common:      common,
		state:       pagerStateBrowse,
		viewport:    vp,
		searchInput: si,
		searchIndex: -1,
	}
	m.initWatcher()
	return m
}

func (m *pagerModel) setSize(w, h int) {
	m.viewport.SetWidth(w)
	m.viewport.SetHeight(h - statusBarHeight)

	if m.showHelp {
		if pagerHelpHeight == 0 {
			pagerHelpHeight = strings.Count(m.helpView(), "\n")
		}
		m.viewport.SetHeight(m.viewport.Height() - (statusBarHeight + pagerHelpHeight))
	}
}

func (m *pagerModel) setContent(s string) {
	m.renderedContent = s
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
	m.clearSearch()
	m.searchInput.Blur()
	m.state = pagerStateBrowse
	m.viewport.SetContent("")
	m.viewport.SetYOffset(0)
	m.unwatchFile()
}

// startSearch clears any previous search and enters search-input mode.
func (m *pagerModel) startSearch() tea.Cmd {
	m.clearSearch()
	m.state = pagerStateSearch
	m.searchInput.Reset()
	return m.searchInput.Focus()
}

// cancelSearch discards the in-progress query and returns to browsing
// without changing any existing highlight state (there is none, since
// startSearch already cleared it).
func (m *pagerModel) cancelSearch() {
	m.searchInput.Blur()
	m.state = pagerStateBrowse
}

// confirmSearch runs the typed query against the current viewport content.
// On success it highlights all matches; on zero matches it shows a status
// message; either way it returns to browse state.
func (m *pagerModel) confirmSearch() tea.Cmd {
	query := m.searchInput.Value()
	m.searchInput.Blur()
	m.state = pagerStateBrowse

	if query == "" {
		return nil
	}

	if !m.runSearch(query) {
		return m.showStatusMessage(pagerStatusMessage{"No matches", false})
	}
	return nil
}

// runSearch finds every occurrence of query in the pristine rendered
// content, stores the results, and bakes highlight styling into the
// viewport content. Returns false if there were no matches, in which case
// search state is left cleared and the viewport shows the plain (unhighlighted)
// rendered content.
func (m *pagerModel) runSearch(query string) bool {
	matches := findMatches(m.renderedContent, query)
	if len(matches) == 0 {
		m.searchQuery = ""
		m.searchMatches = nil
		m.searchIndex = -1
		m.searching = false
		m.viewport.SetContent(m.renderedContent)
		return false
	}

	m.searchQuery = query
	m.searchMatches = matches
	m.searchIndex = nearestMatchIndex(matches, m.viewport.YOffset())
	m.searching = true
	m.applyHighlights()
	return true
}

// applyHighlights rebuilds the viewport content from the pristine rendered
// content with the current search matches baked in via lipgloss.StyleRanges
// (grouping all matches per line into a single StyleRanges call, since
// ranges must be passed in left-to-right, non-overlapping order), then
// scrolls to keep the selected match visible.
func (m *pagerModel) applyHighlights() {
	styles := m.common.styles
	lines := strings.Split(m.renderedContent, "\n")

	rangesByLine := make(map[int][]lipgloss.Range)
	for i, match := range m.searchMatches {
		style := styles.searchHighlightStyle
		if i == m.searchIndex {
			style = styles.searchSelectedHighlightStyle
		}
		rangesByLine[match.line] = append(rangesByLine[match.line], lipgloss.NewRange(match.colStart, match.colEnd, style))
	}
	for line, ranges := range rangesByLine {
		lines[line] = lipgloss.StyleRanges(lines[line], ranges...)
	}

	m.viewport.SetContent(strings.Join(lines, "\n"))
	if m.searchIndex >= 0 {
		sel := m.searchMatches[m.searchIndex]
		m.viewport.EnsureVisible(sel.line, sel.colStart, sel.colEnd)
	}
}

// nearestMatchIndex returns the index of the first match at or after
// yOffset, wrapping to the first match if none qualify.
func nearestMatchIndex(matches []searchMatch, yOffset int) int {
	for i, sm := range matches {
		if sm.line >= yOffset {
			return i
		}
	}
	return 0
}

// clearSearch drops any active search and its highlights.
func (m *pagerModel) clearSearch() {
	m.searchQuery = ""
	m.searchMatches = nil
	m.searchIndex = -1
	m.searching = false
	if m.renderedContent != "" {
		m.viewport.SetContent(m.renderedContent)
	}
}

// reapplySearch re-runs the active search against the current pristine
// rendered content and re-bakes highlights. Used after the document is
// re-rendered (e.g. on terminal resize), since previously computed match
// coordinates don't apply to the new render. If the query no longer matches
// anything, the search is silently dropped (no status message — this is a
// side effect of resizing, not a user-initiated search).
func (m *pagerModel) reapplySearch() {
	if !m.searching {
		return
	}
	m.runSearch(m.searchQuery)
}

// nextMatch selects the next search match, if a search is active.
func (m *pagerModel) nextMatch() {
	if !m.searching || len(m.searchMatches) == 0 {
		return
	}
	m.searchIndex = (m.searchIndex + 1) % len(m.searchMatches)
	m.applyHighlights()
}

// previousMatch selects the previous search match, if a search is active.
func (m *pagerModel) previousMatch() {
	if !m.searching || len(m.searchMatches) == 0 {
		return
	}
	m.searchIndex = (m.searchIndex - 1 + len(m.searchMatches)) % len(m.searchMatches)
	m.applyHighlights()
}

func (m pagerModel) update(msg tea.Msg) (pagerModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	if m.state == pagerStateSearch {
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			switch keyMsg.String() {
			case keyEsc:
				m.cancelSearch()
				return m, nil
			case keyEnter:
				return m, m.confirmSearch()
			}
			var inputCmd tea.Cmd
			m.searchInput, inputCmd = m.searchInput.Update(keyMsg)
			return m, inputCmd
		}
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
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
			m.clearSearch()
			return m, loadLocalMarkdown(&m.currentDocument)

		case "/":
			cmds = append(cmds, m.startSearch())

		case "n":
			m.nextMatch()

		case "N":
			m.previousMatch()

		case "?":
			m.toggleHelp()
		}

	// Glow has rendered the content
	case contentRenderedMsg:
		log.Info("content rendered", "state", m.state)

		m.renderedContent = string(msg)
		if m.searching {
			m.reapplySearch()
		} else {
			m.viewport.SetContent(m.renderedContent)
		}
		cmds = append(cmds, m.watchFile)

	// The file was changed on disk and we're reloading it
	case reloadMsg:
		m.clearSearch()
		return m, loadLocalMarkdown(&m.currentDocument)

	// We've finished editing the document, potentially making changes. Let's
	// retrieve the latest version of the document so that we display
	// up-to-date contents.
	case editorFinishedMsg:
		m.clearSearch()
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

	// Logo, or the search input while a search is being typed
	var logo string
	if m.state == pagerStateSearch {
		logo = m.searchInput.View()
	} else {
		logo = glowLogoView(m.common.styles)
	}

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
	switch {
	case showStatusMessage:
		note = m.statusMessage
	case m.searching:
		query := truncate.StringWithTail(m.searchQuery, maxSearchQueryDisplayWidth, ellipsis)
		if len(m.searchMatches) == 1 {
			note = fmt.Sprintf("Search: %s — 1 match", query)
		} else {
			note = fmt.Sprintf("Search: %s — %d/%d matches", query, m.searchIndex+1, len(m.searchMatches))
		}
	default:
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
		"esc     back to files",
		"q       quit",
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
	s += "\n"
	s += "/        search              n       next match\n"
	s += "                             N       prev match"

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
