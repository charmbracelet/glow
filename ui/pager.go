package ui

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
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
)

type navEntry struct {
	Path    string
	YOffset int
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

	rendered string

	links       []followableLink
	focusedLink int
	history     []navEntry

	pendingRestoreYOffset *int

	watcher     *fsnotify.Watcher
	watchedDir  string
	watchCancel chan struct{}
}

func newPagerModel(common *commonModel) pagerModel {
	vp := viewport.New(0, 0)
	vp.YPosition = 0
	vp.HighPerformanceRendering = common.cfg.HighPerformancePager

	m := pagerModel{
		common:      common,
		state:       pagerStateBrowse,
		viewport:    vp,
		focusedLink: -1,
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

func (m *pagerModel) applyRenderedContent() {
	content := m.rendered
	if m.focusedLink >= 0 {
		content = highlightFocusedLink(content, m.links, m.focusedLink)
	}
	m.setContent(content)
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
	m.viewport.YOffset = 0
	m.rendered = ""
	m.links = nil
	m.focusedLink = -1
	m.history = nil
	m.pendingRestoreYOffset = nil
	m.stopWatching()
}

func (m pagerModel) update(msg tea.Msg) (pagerModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", keyEsc:
			if m.state != pagerStateBrowse {
				m.state = pagerStateBrowse
				return m, nil
			}
		case keyTab:
			if len(m.links) == 0 {
				cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{"No followable links", false}))
				break
			}
			if m.focusedLink < 0 {
				m.focusedLink = 0
			} else {
				m.focusedLink = (m.focusedLink + 1) % len(m.links)
			}
			m.applyRenderedContent()
			cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{"Open: " + m.links[m.focusedLink].ResolvedNote, false}))
		case keyShiftTab, "backtab":
			if len(m.links) == 0 {
				cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{"No followable links", false}))
				break
			}
			if m.focusedLink < 0 {
				m.focusedLink = len(m.links) - 1
			} else {
				m.focusedLink--
				if m.focusedLink < 0 {
					m.focusedLink = len(m.links) - 1
				}
			}
			m.applyRenderedContent()
			cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{"Open: " + m.links[m.focusedLink].ResolvedNote, false}))

		case keyEnter:
			if m.focusedLink >= 0 && m.focusedLink < len(m.links) {
				cmd := m.followFocusedLink()
				return m, cmd
			}
			if len(m.links) > 0 {
				cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{"Tab to select a link", false}))
			}

		case keyBackspace:
			if len(m.history) > 0 {
				cmd := m.goBack()
				return m, cmd
			}
		case "home", "g":
			m.viewport.GotoTop()
			if m.common != nil && m.common.cfg.HighPerformancePager {
				cmds = append(cmds, viewport.Sync(m.viewport))
			}
		case "end", "G":
			m.viewport.GotoBottom()
			if m.common != nil && m.common.cfg.HighPerformancePager {
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
			return m, loadLocalMarkdown(&m.currentDocument)

		case "?":
			m.toggleHelp()
			if m.common != nil && m.common.cfg.HighPerformancePager {
				cmds = append(cmds, viewport.Sync(m.viewport))
			}
		}

	case errMsg:
		m.pendingRestoreYOffset = nil
		cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{msg.Error(), true}))

	// Glow has rendered the content
	case contentRenderedMsg:
		log.Info("content rendered", "state", m.state)

		m.rendered = string(msg)
		m.applyRenderedContent()
		if m.pendingRestoreYOffset != nil {
			m.viewport.YOffset = *m.pendingRestoreYOffset
			if m.viewport.PastBottom() {
				m.viewport.GotoBottom()
			}
			m.pendingRestoreYOffset = nil
		}
		if m.common != nil && m.common.cfg.HighPerformancePager {
			cmds = append(cmds, viewport.Sync(m.viewport))
		}
		cmds = append(cmds, m.startWatching())

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
	rows := [][2]string{
		{"k/↑      up", "g/home  go to top"},
		{"j/↓      down", "G/end   go to bottom"},
		{"b/pgup   page up", "tab     next link"},
		{"f/pgdn   page down", "⇧tab    prev link"},
		{"u        ½ page up", "enter   follow link"},
		{"d        ½ page down", "⌫       go back"},
		{"", "c       copy contents"},
		{"", "e       edit this document"},
		{"", "r       reload this document"},
		{"", "esc     back to files"},
		{"", "q       quit"},
	}

	s += "\n"
	for _, row := range rows {
		left := row[0]
		right := row[1]
		if left != "" {
			left = fmt.Sprintf("%-24s", left)
		} else {
			left = strings.Repeat(" ", 24)
		}
		s += left + right + "\n"
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

func (m *pagerModel) startWatching() tea.Cmd {
	if m.watcher == nil || m.currentDocument.localPath == "" {
		return nil
	}

	m.stopWatching()

	dir := m.localDir()
	if err := m.watcher.Add(dir); err != nil {
		log.Error("error adding dir to fsnotify watcher", "error", err)
		return nil
	}
	m.watchedDir = dir
	m.watchCancel = make(chan struct{})

	cancel := m.watchCancel
	return func() tea.Msg { return m.watchFile(cancel) }
}

func (m *pagerModel) watchFile(cancel <-chan struct{}) tea.Msg {
	log.Info("fsnotify watching dir", "dir", m.watchedDir)

	for {
		select {
		case <-cancel:
			return nil
		case event, ok := <-m.watcher.Events:
			if !ok {
				return nil
			}
			if event.Name != m.currentDocument.localPath {
				continue
			}

			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
				continue
			}

			log.Debug("fsnotify event", "file", event.Name, "event", event.Op)
			return reloadMsg{}
		case err, ok := <-m.watcher.Errors:
			if !ok {
				return nil
			}
			log.Debug("fsnotify error", "dir", m.watchedDir, "error", err)
		}
	}
}

func (m *pagerModel) stopWatching() {
	if m.watchCancel != nil {
		close(m.watchCancel)
		m.watchCancel = nil
	}

	if m.watcher == nil || m.watchedDir == "" {
		return
	}

	err := m.watcher.Remove(m.watchedDir)
	if err == nil {
		log.Debug("fsnotify dir unwatched", "dir", m.watchedDir)
	} else {
		log.Error("fsnotify fail to unwatch dir", "dir", m.watchedDir, "error", err)
	}
	m.watchedDir = ""
}

func (m *pagerModel) localDir() string {
	return filepath.Dir(m.currentDocument.localPath)
}

func (m *pagerModel) followFocusedLink() tea.Cmd {
	l := m.links[m.focusedLink]
	if l.ResolvedPath == "" {
		return nil
	}
	if m.currentDocument.localPath != "" {
		m.history = append(m.history, navEntry{Path: m.currentDocument.localPath, YOffset: m.viewport.YOffset})
	}

	m.focusedLink = -1
	m.viewport.GotoTop()
	m.pendingRestoreYOffset = nil

	md := &markdown{
		localPath: l.ResolvedPath,
		Note:      l.ResolvedNote,
	}
	return loadLocalMarkdown(md)
}

func (m *pagerModel) goBack() tea.Cmd {
	if len(m.history) == 0 {
		return nil
	}

	last := m.history[len(m.history)-1]
	m.history = m.history[:len(m.history)-1]

	m.focusedLink = -1
	y := last.YOffset
	m.pendingRestoreYOffset = &y
	m.viewport.GotoTop()

	md := &markdown{
		localPath: last.Path,
		Note:      stripAbsolutePath(last.Path, m.common.cwd),
	}
	return loadLocalMarkdown(md)
}
