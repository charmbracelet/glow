package ui

import (
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	glamansi "charm.land/glamour/v2/ansi"
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

// RemoteImageNotLoadedNote is appended to the URL of a remote image when
// loading remote images is disabled, pointing at the config setting that
// loads them.
const RemoteImageNotLoadedNote = ` (not loaded. Set "loadRemoteImages: true" in your glow config to load remote images)`

// remoteImagePattern matches inline markdown images, i.e.
// ![alt](https://...), and HTML ones, i.e. <img src="https://...">, that are
// fetched over the network.
var remoteImagePattern = regexp.MustCompile(`(?i)!\[[^\]]*\]\(\s*["']?https?://|<img[^>]+src\s*=\s*["']?https?://`)

var (
	// referenceImagePattern matches markdown images that point at a link
	// reference, i.e. ![alt][badge].
	referenceImagePattern = regexp.MustCompile(`!\[[^\]]*\]\[([^\]]*)\]`)
	// referenceDefinitionPattern matches the definition of a link
	// reference, i.e. [badge]: https://example.com/badge.svg.
	referenceDefinitionPattern = regexp.MustCompile(`(?m)^ {0,3}\[([^\]]+)\]:\s*<?(\S+)`)
)

// hasRemoteImages reports whether the markdown references any image by
// http(s) URL, i.e. whether rendering it involves fetching images.
func hasRemoteImages(md string) bool {
	if remoteImagePattern.MatchString(md) {
		return true
	}

	// Images can also point at a link reference, e.g. a badge in a README:
	// ![build][badge] with [badge]: https://example.com/badge.svg. Match the
	// references that are known to be remote; link labels are
	// case-insensitive.
	var remoteReferences map[string]bool
	for _, m := range referenceDefinitionPattern.FindAllStringSubmatch(md, -1) {
		if isRemoteURL(m[2]) {
			if remoteReferences == nil {
				remoteReferences = make(map[string]bool)
			}
			remoteReferences[strings.ToLower(m[1])] = true
		}
	}
	if len(remoteReferences) == 0 {
		return false
	}
	for _, m := range referenceImagePattern.FindAllStringSubmatch(md, -1) {
		if remoteReferences[strings.ToLower(m[1])] {
			return true
		}
	}
	return false
}

// isRemoteURL reports whether the given URL is served over http(s).
func isRemoteURL(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}

var pagerHelpHeight int

type (
	// contentRenderedMsg is sent when the glamour rendering of a document
	// finishes. graphics holds the graphics protocol sequences that must be
	// written to the terminal before the content, which references them via
	// unicode placeholders, is displayed. body is the markdown that was
	// rendered, and remoteImagesPending reports that it was rendered without
	// its remote images, which the pager then loads in a second pass.
	contentRenderedMsg struct {
		content             string
		graphics            []string
		body                string
		remoteImagesPending bool
	}
	// remoteImagesLoadedMsg is sent when the second rendering pass, which
	// loads the document's remote images, finishes. token identifies the
	// pass, so a result that was superseded, e.g. by a resize, is ignored.
	remoteImagesLoadedMsg struct {
		content  string
		graphics []string
		token    int
	}
	reloadMsg struct{}
)

type pagerState int

const (
	pagerStateBrowse pagerState = iota
	pagerStateStatusMessage
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

	// pendingContent is content that is waiting on its graphics protocol
	// sequences to be written to the terminal before it can be displayed.
	// See the contentRenderedMsg handling in update.
	pendingContent string

	// remoteImagesLoading reports whether the remote images of the current
	// document are still being fetched, and remoteSpinner spins while they
	// are; the text is displayed without them in the meantime.
	// remoteImagesPass identifies the running fetching pass, so that its
	// result can be told apart from those of older passes.
	remoteImagesLoading bool
	remoteImagesPass    int
	remoteSpinner       spinner.Model

	watcher *fsnotify.Watcher
}

func newPagerModel(common *commonModel) pagerModel {
	// Init viewport
	vp := viewport.New()

	sp := spinner.New()
	sp.Spinner = spinner.Line
	sp.Style = common.styles.stashSpinnerStyle

	m := pagerModel{
		common:        common,
		state:         pagerStateBrowse,
		viewport:      vp,
		remoteSpinner: sp,
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
	m.remoteImagesLoading = false
	m.viewport.SetContent("")
	m.viewport.SetYOffset(0)
	m.unwatchFile()
}

func (m pagerModel) update(msg tea.Msg) (pagerModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

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
			return m, loadLocalMarkdown(&m.currentDocument)

		case "?":
			m.toggleHelp()
		}

	// Glow has rendered the content
	case contentRenderedMsg:
		log.Info("content rendered", "state", m.state)

		if msg.remoteImagesPending {
			// The remote images are still being fetched; the content shown
			// in the meantime is the text without them. Start the second
			// rendering pass that loads and renders them.
			m.remoteImagesLoading = true
			m.remoteImagesPass++
			cmds = append(cmds, m.remoteSpinner.Tick,
				loadRemoteImagesCmd(m, msg.body, m.remoteImagesPass))
		}

		if len(msg.graphics) > 0 {
			// The content references images via unicode placeholders, so the
			// graphics sequences must be written to the terminal before it
			// can be displayed. RawMsg is written out-of-band right before
			// this model sees it, so stash the content and wait for it.
			m.pendingContent = msg.content
			return m, tea.Batch(append(cmds, tea.Raw(strings.Join(msg.graphics, "")))...)
		}

		m.setContent(msg.content)
		cmds = append(cmds, m.watchFile)

	// The remote images of the current document have been fetched and
	// rendered; swap in the content that displays them.
	case remoteImagesLoadedMsg:
		if !m.remoteImagesLoading || msg.token != m.remoteImagesPass {
			// A newer rendering pass took over, or the user left the
			// document while its images were loading; keep what's on screen.
			return m, nil
		}

		m.remoteImagesLoading = false
		if len(msg.graphics) > 0 {
			m.pendingContent = msg.content
			return m, tea.Raw(strings.Join(msg.graphics, ""))
		}
		m.setContent(msg.content)

	// Keep the loading indicator spinning while the remote images load.
	case spinner.TickMsg:
		if m.remoteImagesLoading {
			var cmd tea.Cmd
			m.remoteSpinner, cmd = m.remoteSpinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	// The graphics sequences for the pending content have been written
	case tea.RawMsg:
		if m.pendingContent != "" {
			m.setContent(m.pendingContent)
			m.pendingContent = ""
			cmds = append(cmds, m.watchFile)
		}

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
	switch {
	case m.remoteImagesLoading:
		note = m.remoteSpinner.View() + " Loading remote images..."
	case showStatusMessage:
		note = m.statusMessage
	default:
		note = m.currentDocument.Note
	}
	note = truncate.StringWithTail(" "+note+" ", uint(max(0, //nolint:gosec
		m.common.width-
			ansi.PrintableRuneWidth(logo)-
			ansi.PrintableRuneWidth(scrollPercent)-
			ansi.PrintableRuneWidth(helpNote),
	)), ellipsis)
	// The loading indicator takes the place of the status message, so it's
	// styled like the file name rather than like a message.
	showMessage := showStatusMessage && !m.remoteImagesLoading
	if showMessage {
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
	if showMessage {
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

// renderWithGlamour renders the document for the pager. When remote images
// are enabled and the document references any, the text is rendered without
// them first so it can be displayed right away: fetching them takes a while,
// and the document would otherwise stay off the screen until it's done. The
// pager then starts a second pass that loads and renders the images, see the
// contentRenderedMsg handling in update.
func renderWithGlamour(m pagerModel, md string) tea.Cmd {
	pending := remoteImagesEnabled(m) && hasRemoteImages(md)

	// Images that don't need fetching are loaded in the first pass.
	return renderContent(m, md, m.common.cfg.LoadRemoteImages && !pending, pending)
}

// renderContent renders a document with glamour and reports the result to
// the pager. remote loads images referenced by http(s) URLs; pending marks a
// render that leaves the remote images to a follow-up pass.
func renderContent(m pagerModel, md string, remote, pending bool) tea.Cmd {
	return func() tea.Msg {
		s, graphics, err := glamourRender(m, md, remote)
		if err != nil {
			log.Error("error rendering with Glamour", "error", err)
			return errMsg{err}
		}
		return contentRenderedMsg{
			content:             s,
			graphics:            graphics,
			body:                md,
			remoteImagesPending: pending,
		}
	}
}

// remoteImagesEnabled reports whether remote images should be loaded when
// rendering the current document.
func remoteImagesEnabled(m pagerModel) bool {
	return m.common.cfg.LoadRemoteImages && glamourImages(m) &&
		utils.IsMarkdownFile(m.currentDocument.Note)
}

// loadRemoteImagesCmd renders a document a second time, fetching and
// rendering its remote images. token identifies the pass, so a result that a
// newer pass has superseded can be discarded.
func loadRemoteImagesCmd(m pagerModel, md string, token int) tea.Cmd {
	return func() tea.Msg {
		s, graphics, err := glamourRender(m, md, true)
		if err != nil {
			log.Error("error loading remote images", "error", err)
			return errMsg{err}
		}
		return remoteImagesLoadedMsg{content: s, graphics: graphics, token: token}
	}
}

// glamourImages reports whether to render images in the pager using the
// terminal's graphics protocol.
func glamourImages(m pagerModel) bool {
	return m.common.cfg.GlamourEnabled && m.common.cfg.Images &&
		m.common.imageProtocol != glamansi.ImageProtocolNone
}

// This is where the magic happens.
func glamourRender(m pagerModel, markdown string, loadRemoteImages bool) (string, []string, error) {
	images := glamourImages(m)
	trunc := lipgloss.NewStyle().MaxWidth(m.viewport.Width() - lineNumberWidth).Render

	if !m.common.cfg.GlamourEnabled {
		return markdown, nil, nil
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

	if images && !isCode {
		if width == 0 {
			// Without a wrap width, images would be sized to their
			// natural dimensions, which can be far larger than the
			// viewport.
			width = m.viewport.Width()
			options[1] = glamour.WithWordWrap(width)
		}
		if m.common.cfg.ShowLineNumbers {
			// Leave room for the line number gutter so the rendered
			// content, including image placeholder grids, doesn't get
			// truncated.
			width = max(0, width-lineNumberWidth)
			options[1] = glamour.WithWordWrap(width)
		}
		options = append(options,
			glamour.WithImageProtocol(m.common.imageProtocol),
			glamour.WithMaxImageSize(0, m.common.cfg.ImageMaxRows),
		)
		switch {
		case loadRemoteImages:
			options = append(options, glamour.WithRemoteImages())
		case m.common.cfg.LoadRemoteImages:
			// This is the text-only render that precedes the remote image
			// loading one, so don't claim the images were not loaded.
			options = append(options, glamour.WithRemoteImageNotLoadedNote(""))
		default:
			options = append(options, glamour.WithRemoteImageNotLoadedNote(RemoteImageNotLoadedNote))
		}
	}

	if m.common.cfg.PreserveNewLines {
		options = append(options, glamour.WithPreservedNewLines())
	}
	if m.currentDocument.localPath != "" {
		// Resolve image URLs relative to the document's directory, the
		// same way the CLI does.
		options = append(options, glamour.WithBaseURL(utils.FileBaseURL(m.currentDocument.localPath)))
	}
	r, err := glamour.NewTermRenderer(options...)
	if err != nil {
		return "", nil, fmt.Errorf("error creating glamour renderer: %w", err)
	}

	if isCode {
		markdown = utils.WrapCodeBlock(markdown, filepath.Ext(m.currentDocument.Note))
	}

	out, err := r.Render(markdown)
	if err != nil {
		return "", nil, fmt.Errorf("error rendering markdown: %w", err)
	}

	// The graphics commands are out-of-band sequences the caller must write
	// to the terminal before displaying the rendered content.
	graphics := r.GraphicsCommands()

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

	return content.String(), graphics, nil
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
