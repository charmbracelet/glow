package ui

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/keygen"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/muesli/gitcha"
	te "github.com/muesli/termenv"
)

const (
	noteCharacterLimit   = 256             // should match server
	statusMessageTimeout = time.Second * 2 // how long to show status messages like "stashed!"
)

var (
	config            Config
	glowLogoTextColor = common.Color("#ECFD65")
	debug             = false // true if we're logging to a file, in which case we'll log more stuff
)

// Config contains configuration specified to the TUI.
type Config struct {
	// For debugging the UI
	Logfile              string `env:"GLOW_UI_LOGFILE"`
	HighPerformancePager bool   `env:"GLOW_UI_HIGH_PERFORMANCE_PAGER" default:"true"`
	GlamourEnabled       bool   `env:"GLOW_UI_ENABLE_GLAMOUR" default:"true"`
}

// NewProgram returns a new Tea program.
func NewProgram(style string, cfg Config) *tea.Program {
	if cfg.Logfile != "" {
		log.Println("-- Starting Glow ----------------")
		log.Printf("High performance pager: %v", cfg.HighPerformancePager)
		log.Printf("Glamour rendering: %v", cfg.GlamourEnabled)
		log.Println("Bubble Tea now initializing...")
		debug = true
	}
	config = cfg
	return tea.NewProgram(initialize(cfg, style), update, view)
}

// MESSAGES

type errMsg struct{ err error }
type newCharmClientMsg *charm.Client
type sshAuthErrMsg struct{}
type keygenFailedMsg struct{ err error }
type keygenSuccessMsg struct{}
type initLocalFileSearchMsg struct {
	cwd string
	ch  chan gitcha.SearchResult
}
type foundLocalFileMsg gitcha.SearchResult
type localFileSearchFinished struct{}
type gotStashMsg []*charm.Markdown
type stashLoadErrMsg struct{ err error }
type gotNewsMsg []*charm.Markdown
type statusMessageTimeoutMsg applicationContext
type newsLoadErrMsg struct{ err error }

func (e errMsg) Error() string          { return e.err.Error() }
func (k keygenFailedMsg) Error() string { return k.err.Error() }
func (s stashLoadErrMsg) Error() string { return s.err.Error() }
func (s newsLoadErrMsg) Error() string  { return s.err.Error() }

// MODEL

// Which part of the application something appies to. Occasionally used as an
// arguments to command and messages.
type applicationContext int

const (
	stashContext applicationContext = iota
	pagerContext
)

type state int

const (
	stateShowStash state = iota
	stateShowDocument
)

// String translates the staus to a human-readable string. This is just for
// debugging.
func (s state) String() string {
	return [...]string{
		"showing stash",
		"showing document",
	}[s]
}

type keygenState int

const (
	keygenUnstarted keygenState = iota
	keygenRunning
	keygenFinished
)

type model struct {
	cfg            Config
	cc             *charm.Client
	keygenState    keygenState
	state          state
	fatalErr       error
	stash          stashModel
	pager          pagerModel
	terminalWidth  int
	terminalHeight int
	cwd            string // directory from which we're running Glow

	// Channel that receives paths to local markdown files
	// (via the github.com/muesli/gitcha package)
	localFileFinder chan gitcha.SearchResult
}

// unloadDocument unloads a document from the pager. Note that while this
// method alters the model we also need to send along any commands returned.
func (m *model) unloadDocument() []tea.Cmd {
	m.state = stateShowStash
	m.stash.state = stashStateReady
	m.pager.unload()
	m.pager.showHelp = false

	var batch []tea.Cmd
	if m.pager.viewport.HighPerformanceRendering {
		batch = append(batch, tea.ClearScrollArea)
	}
	if !m.stash.loaded.done() || m.stash.loadingFromNetwork {
		batch = append(batch, spinner.Tick(m.stash.spinner))
	}
	return batch
}

// INIT

func initialize(cfg Config, style string) func() (tea.Model, tea.Cmd) {
	return func() (tea.Model, tea.Cmd) {
		if style == "auto" {
			dbg := te.HasDarkBackground()
			if dbg {
				style = "dark"
			} else {
				style = "light"
			}
		}

		m := model{
			cfg:         cfg,
			stash:       newStashModel(),
			pager:       newPagerModel(style),
			state:       stateShowStash,
			keygenState: keygenUnstarted,
		}

		return m, tea.Batch(
			findLocalFiles,
			newCharmClient,
			spinner.Tick(m.stash.spinner),
		)
	}
}

// UPDATE

func update(msg tea.Msg, mdl tea.Model) (tea.Model, tea.Cmd) {
	m, ok := mdl.(model)
	if !ok {
		return model{
			fatalErr: errors.New("could not perform assertion on model in update"),
		}, tea.Quit
	}

	// If there's been an error, any key exits
	if m.fatalErr != nil {
		if _, ok := msg.(tea.KeyMsg); ok {
			return m, tea.Quit
		}
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			var cmd tea.Cmd

			// Send these keys through to stash
			switch m.state {
			case stateShowStash:

				switch m.stash.state {
				case stashStateSettingNote, stashStatePromptDelete, stashStateShowingError:
					m.stash, cmd = stashUpdate(msg, m.stash)
					return m, cmd
				}

			// Special cases for the pager
			case stateShowDocument:
				switch m.pager.state {
				// If setting a note send all keys straight through
				case pagerStateSetNote:
					var batch []tea.Cmd
					newPagerModel, cmd := pagerUpdate(msg, m.pager)
					m.pager = newPagerModel
					batch = append(batch, cmd)
					return m, tea.Batch(batch...)

				// Otherwise let the user exit the view or application as
				// normal.
				default:
					switch msg.String() {
					case "q":
						return m, tea.Quit
					case "esc":
						var batch []tea.Cmd
						batch = m.unloadDocument()
						return m, tea.Batch(batch...)
					}
				}
			}

			return m, tea.Quit

		case "left", "h", "delete":
			if m.state == stateShowDocument && m.pager.state != pagerStateSetNote {
				cmds = append(cmds, m.unloadDocument()...)
				return m, tea.Batch(cmds...)
			}

		// Ctrl+C always quits no matter where in the application you are.
		case "ctrl+c":
			return m, tea.Quit

		// Repaint
		case "ctrl+l":
			// TODO
			return m, nil
		}

	// Window size is received when starting up and on every resize
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height
		m.stash.setSize(msg.Width, msg.Height)
		m.pager.setSize(msg.Width, msg.Height)

		// TODO: load more stash pages if we've resized, are on the last page,
		// and haven't loaded more pages yet.

	// We've started looking for local files
	case initLocalFileSearchMsg:
		m.localFileFinder = msg.ch
		m.cwd = msg.cwd
		cmds = append(cmds, findNextLocalFile(m))

	// We found a local file
	case foundLocalFileMsg:
		newMd := localFileToMarkdown(m.cwd, gitcha.SearchResult(msg))
		m.stash.addMarkdowns(newMd)
		cmds = append(cmds, findNextLocalFile(m))

	case sshAuthErrMsg:
		// If we haven't run the keygen yet, do that
		if m.keygenState != keygenFinished {
			m.keygenState = keygenRunning
			cmds = append(cmds, generateSSHKeys)
		} else {
			// The keygen ran but things still didn't work and we can't auth
			m.stash.err = errors.New("SSH authentication failed; we tried ssh-agent, loading keys from disk, and generating SSH keys")

			// Even though it failed, news/stash loading is finished
			m.stash.loaded |= loadedStash | loadedNews
			m.stash.loadingFromNetwork = false
		}

	case keygenFailedMsg:
		// Keygen failed. That sucks.
		m.stash.err = errors.New("could not authenticate; could not generate SSH keys")
		m.keygenState = keygenFinished

		// Even though it failed, news/stash loading is finished
		m.stash.loaded |= loadedStash | loadedNews
		m.stash.loadingFromNetwork = false

	case keygenSuccessMsg:
		// The keygen's done, so let's try initializing the charm client again
		m.keygenState = keygenFinished
		cmds = append(cmds, newCharmClient)

	case newCharmClientMsg:
		m.cc = msg
		m.stash.cc = msg
		m.pager.cc = msg
		cmds = append(cmds, loadStash(m.stash), loadNews(m.stash))

	case fetchedMarkdownMsg:
		m.pager.currentDocument = *msg
		cmds = append(cmds, renderWithGlamour(m.pager, msg.Body))

	case contentRenderedMsg:
		m.state = stateShowDocument

	case noteSavedMsg:
		// A note was saved to a document. This will have be done in the
		// pager, so we'll need to find the corresponding note in the stash.
		// So, pass the message to the stash for processing.
		stashModel, cmd := stashUpdate(msg, m.stash)
		m.stash = stashModel
		return m, cmd

	case localFileSearchFinished, gotStashMsg, gotNewsMsg:
		// Also pass these messages to the stash so we can keep it updated
		// about network activity.
		stashModel, cmd := stashUpdate(msg, m.stash)
		m.stash = stashModel
		return m, cmd

	case stashSuccessMsg:
		// Something was stashed. Update the stash listing but don't run an
		// actual update on the stash since we don't want to trigger the status
		// message and generally don't want any other effects.
		if m.state == stateShowDocument {
			md := markdown(msg)
			_ = m.stash.removeLocalMarkdown(md.localPath)
			m.stash.addMarkdowns(&md)
		}
	}

	// Process children
	switch m.state {
	case stateShowStash:
		newStashModel, cmd := stashUpdate(msg, m.stash)
		m.stash = newStashModel
		cmds = append(cmds, cmd)

	case stateShowDocument:
		newPagerModel, cmd := pagerUpdate(msg, m.pager)
		m.pager = newPagerModel
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// VIEW

func view(mdl tea.Model) string {
	m, ok := mdl.(model)
	if !ok {
		return "could not perform assertion on model in view"
	}

	if m.fatalErr != nil {
		return errorView(m.fatalErr, true)
	}

	switch m.state {
	case stateShowDocument:
		return pagerView(m.pager)
	default:
		return stashView(m.stash)
	}
}

func errorView(err error, fatal bool) string {
	exitMsg := "press any key to "
	if fatal {
		exitMsg += "exit"
	} else {
		exitMsg += "return"
	}
	s := fmt.Sprintf("%s\n\n%v\n\n%s",
		te.String(" ERROR ").
			Foreground(common.Cream.Color()).
			Background(common.Red.Color()).
			String(),
		err,
		common.Subtle(exitMsg),
	)
	return "\n" + indent(s, 3)
}

// COMMANDS

func findLocalFiles() tea.Msg {
	cwd, err := os.Getwd()
	if err != nil {
		if debug {
			log.Println("error finding local files:", err)
		}
		return errMsg{err}
	}

	ch, err := gitcha.FindFiles(cwd, []string{"*.md"})
	if err != nil {
		if debug {
			log.Println("error finding local files:", err)
		}
		return errMsg{err}
	}

	return initLocalFileSearchMsg{ch: ch, cwd: cwd}
}

func findNextLocalFile(m model) tea.Cmd {
	return func() tea.Msg {
		pathStr, ok := <-m.localFileFinder
		if ok {
			// Okay now find the next one
			return foundLocalFileMsg(pathStr)
		}
		// We're done
		if debug {
			log.Println("Local file search finished")
		}
		return localFileSearchFinished{}
	}
}

func newCharmClient() tea.Msg {
	cfg, err := charm.ConfigFromEnv()
	if err != nil {
		return errMsg{err}
	}

	cc, err := charm.NewClient(cfg)
	if err == charm.ErrMissingSSHAuth {
		if debug {
			log.Println("missing SSH auth:", err)
		}
		return sshAuthErrMsg{}
	} else if err != nil {
		if debug {
			log.Println("error creating new charm client:", err)
		}
		return errMsg{err}
	}

	return newCharmClientMsg(cc)
}

func loadStash(m stashModel) tea.Cmd {
	return func() tea.Msg {
		if m.cc == nil {
			err := errors.New("no charm client")
			if debug {
				log.Println("error loading stash:", err)
			}
			return stashLoadErrMsg{err}
		}
		stash, err := m.cc.GetStash(m.page)
		if err != nil {
			if debug {
				log.Println("error loading stash:", err)
			}
			return stashLoadErrMsg{err}
		}
		return gotStashMsg(stash)
	}
}

func loadNews(m stashModel) tea.Cmd {
	return func() tea.Msg {
		if m.cc == nil {
			err := errors.New("no charm client")
			if debug {
				log.Println("error loading news:", err)
			}
			return newsLoadErrMsg{err}
		}
		news, err := m.cc.GetNews(1) // just fetch the first page
		if err != nil {
			if debug {
				log.Println("error loading news:", err)
			}
			return newsLoadErrMsg{err}
		}
		return gotNewsMsg(news)
	}
}

func generateSSHKeys() tea.Msg {
	_, err := keygen.NewSSHKeyPair(nil)
	if err != nil {
		if debug {
			log.Println("keygen failed:", err)
		}
		return keygenFailedMsg{err}
	}
	return keygenSuccessMsg{}
}

func saveDocumentNote(cc *charm.Client, id int, note string) tea.Cmd {
	if cc == nil {
		return func() tea.Msg {
			err := errors.New("can't set note; no charm client")
			if debug {
				log.Println("error saving note:", err)
			}
			return errMsg{err}
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
					log.Println("error stashing document:", err)
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

func waitForStatusMessageTimeout(appCtx applicationContext, t *time.Timer) tea.Cmd {
	return func() tea.Msg {
		<-t.C
		return statusMessageTimeoutMsg(appCtx)
	}
}

// ETC

// Convert local file path to Markdown. Note that we could be doing things
// like checking if the file is a directory, but we trust that gitcha has
// already done that.
func localFileToMarkdown(cwd string, res gitcha.SearchResult) *markdown {
	md := &markdown{
		markdownType: localMarkdown,
		localPath:    res.Path,
		displayPath:  strings.Replace(res.Path, cwd+"/", "", -1), // strip absolute path
		Markdown:     charm.Markdown{},
	}

	md.Markdown.Note = md.displayPath
	md.CreatedAt = res.Info.ModTime() // last modified time

	return md
}

// Lightweight version of reflow's indent function.
func indent(s string, n int) string {
	if n <= 0 || s == "" {
		return s
	}
	l := strings.Split(s, "\n")
	b := strings.Builder{}
	i := strings.Repeat(" ", n)
	for _, v := range l {
		fmt.Fprintf(&b, "%s%s\n", i, v)
	}
	return b.String()
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
