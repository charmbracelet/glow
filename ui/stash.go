package ui

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/dustin/go-humanize"
	"github.com/lithammer/fuzzysearch/fuzzy"
	runewidth "github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/ansi"
	te "github.com/muesli/termenv"
)

const (
	stashIndent                = 1
	stashViewItemHeight        = 3
	stashViewTopPadding        = 5
	stashViewBottomPadding     = 4
	stashViewHorizontalPadding = 6
	setNotePromptText          = "Memo: "
	searchNotePromptText       = "Search: "
)

var (
	stashHelpItemStyle func(string) string = te.Style{}.Foreground(common.NewColorPair("#5C5C5C", "#9B9B9B").Color()).Styled
	dividerDot         string              = te.String(" • ").Foreground(common.NewColorPair("#3C3C3C", "#DDDADA").Color()).String()
	offlineHeaderNote  string              = te.String("(Offline)").Foreground(common.NewColorPair("#3C3C3C", "#DDDADA").Color()).String()
)

// MSG

type fetchedMarkdownMsg *markdown
type deletedStashedItemMsg int

// MODEL

type DocumentType byte

const (
	LocalDocuments DocumentType = 1 << iota
	StashedDocuments
	NewsDocuments
)

type stashState int

const (
	stashStateReady stashState = iota
	stashStatePromptDelete
	stashStateLoadingDocument
	stashStateSettingNote
	stashStateShowingError
	stashStateSearchNotes
	stashStateShowFiltered
)

type stashModel struct {
	cc                 *charm.Client
	cfg                *Config
	authStatus         authStatus
	state              stashState
	err                error
	markdowns          []*markdown
	spinner            spinner.Model
	noteInput          textinput.Model
	searchInput        textinput.Model
	terminalWidth      int
	terminalHeight     int
	stashFullyLoaded   bool         // have we loaded all available stashed documents from the server?
	loadingFromNetwork bool         // are we currently loading something from the network?
	loaded             DocumentType // load status for news, stash and local files loading; we find out exactly with bitmasking

	// Paths to files being stashed. We treat this like a set, ignoring the
	// value portion with an empty struct.
	filesStashing map[string]struct{}

	// This is just the index of the current page in view. To get the index
	// of the selected item as it relates to the full set of documents we've
	// fetched use the mardownIndex() method of this struct.
	index int

	// This handles the local pagination, which is different than the page
	// we're fetching from on the server side
	paginator paginator.Model

	// Page we're fetching stash items from on the server, which is different
	// from the local pagination. Generally, the server will return more items
	// than we can display at a time so we can paginate locally without having
	// to fetch every time.
	page int

	showStatusMessage  bool
	statusMessage      string
	statusMessageTimer *time.Timer
}

func (m stashModel) localOnly() bool {
	return m.cfg.DocumentTypes == LocalDocuments
}

func (m stashModel) stashedOnly() bool {
	return m.cfg.DocumentTypes&LocalDocuments == 0
}

func (m stashModel) loadingDone() bool {
	// Do the types loaded match the types we want to have?
	return m.loaded == m.cfg.DocumentTypes
}

func (m *stashModel) setSize(width, height int) {
	m.terminalWidth = width
	m.terminalHeight = height

	// Update the paginator
	m.setTotalPages()

	m.noteInput.Width = m.terminalWidth - stashViewHorizontalPadding*2 - len(setNotePromptText)
	m.searchInput.Width = m.terminalWidth - stashViewHorizontalPadding*2 - len(searchNotePromptText)
}

// Sets the total paginator pages according to the amount of markdowns for the
// current state.
func (m *stashModel) setTotalPages() {
	m.paginator.PerPage = max(1, (m.terminalHeight-stashViewTopPadding-stashViewBottomPadding)/stashViewItemHeight)

	if pages := len(m.getNotes()); pages < 1 {
		m.paginator.SetTotalPages(1)
	} else {
		m.paginator.SetTotalPages(pages)
	}

	// Make sure the page stays in bounds
	if m.paginator.Page >= m.paginator.TotalPages-1 {
		m.paginator.Page = max(0, m.paginator.TotalPages-1)
	}
}

// MarkdownIndex returns the index of the currently selected markdown item.
func (m stashModel) markdownIndex() int {
	return m.paginator.Page*m.paginator.PerPage + m.index
}

// Return the current selected markdown in the stash.
func (m stashModel) selectedMarkdown() *markdown {
	i := m.markdownIndex()
	markdowns := m.getNotes()

	if i < 0 || len(markdowns) == 0 || len(markdowns) <= i {
		return nil
	}

	return markdowns[i]
}

// Adds markdown documents to the model.
func (m *stashModel) addMarkdowns(mds ...*markdown) {
	if len(mds) > 0 {
		m.markdowns = append(m.markdowns, mds...)
		sort.Sort(markdownsByLocalFirst(m.markdowns))
		m.setTotalPages()
	}
}

// Find a local markdown by its path and remove it.
func (m *stashModel) removeLocalMarkdown(localPath string) error {
	i := -1

	// Look for local markdown
	for j, doc := range m.markdowns {
		if doc.localPath == localPath {
			i = j
			break
		}
	}

	// Did we find it?
	if i == -1 {
		err := fmt.Errorf("could't find local markdown %s; not removing from stash", localPath)
		if debug {
			log.Println(err)
		}
		return err
	}

	// Slice out markdown
	if i >= 0 {
		m.markdowns = append(m.markdowns[:i], m.markdowns[i+1:]...)
	}
	return nil
}

// Return the number of markdown documents of a given type.
func (m stashModel) countMarkdowns(t markdownType) (found int) {
	if len(m.getNotes()) == 0 {
		return
	}
	for i := 0; i < len(m.getNotes()); i++ {
		if m.getNotes()[i].markdownType == t {
			found++
		}
	}
	return
}

// Returns the stashed markdown notes. When the model state indicates that
// filtering is desired, this also filters and ranks the notes by the search
// term in the searchInput field.
func (m *stashModel) getNotes() []*markdown {
	if m.searchInput.Value() == "" {
		return m.markdowns
	}
	if m.state != stashStateSearchNotes &&
		m.state != stashStateShowFiltered &&
		m.state != stashStatePromptDelete &&
		m.state != stashStateSettingNote {
		return m.markdowns
	}

	targets := []string{}

	for _, t := range m.markdowns {
		note := ""
		switch t.markdownType {
		case newsMarkdown:
			note = "News: " + t.Note
		case stashedMarkdown:
			note = t.Note
		default:
			note = t.localPath
		}

		targets = append(targets, note)
	}

	ranks := fuzzy.RankFindFold(m.searchInput.Value(), targets)
	sort.Sort(ranks)

	filtered := []*markdown{}
	for _, r := range ranks {
		filtered = append(filtered, m.markdowns[r.OriginalIndex])
	}

	return filtered
}

func (m *stashModel) hideStatusMessage() {
	m.showStatusMessage = false
	if m.statusMessageTimer != nil {
		m.statusMessageTimer.Stop()
	}
}

func (m *stashModel) moveCursorUp() {
	m.index--
	if m.index < 0 && m.paginator.Page == 0 {
		// Stop
		m.index = 0
		return
	}

	if m.index >= 0 {
		return
	}
	// Go to previous page
	m.paginator.PrevPage()

	m.index = m.paginator.ItemsOnPage(len(m.getNotes())) - 1
}

func (m *stashModel) moveCursorDown() {
	itemsOnPage := m.paginator.ItemsOnPage(len(m.getNotes()))

	m.index++
	if m.index < itemsOnPage {
		return
	}

	if !m.paginator.OnLastPage() {
		m.paginator.NextPage()
		m.index = 0
		return
	}

	// during filtering the cursor position can exceed the number of
	// itemsOnPage. It's more intuitive to start the cursor at the
	// topmost position when moving it down in this scenario.
	if m.index > itemsOnPage {
		m.index = 0
		return
	}
	m.index = itemsOnPage - 1
}

// INIT

func newStashModel(cfg *Config, as authStatus) stashModel {
	sp := spinner.NewModel()
	sp.Frames = spinner.Line
	sp.ForegroundColor = common.SpinnerColor.String()
	sp.HideFor = time.Millisecond * 50
	sp.MinimumLifetime = time.Millisecond * 180
	sp.Start()

	p := paginator.NewModel()
	p.Type = paginator.Dots
	p.InactiveDot = common.Subtle("•")

	ni := textinput.NewModel()
	ni.Prompt = te.String(setNotePromptText).Foreground(common.YellowGreen.Color()).String()
	ni.CursorColor = common.Fuschia.String()
	ni.CharLimit = noteCharacterLimit
	ni.Focus()

	si := textinput.NewModel()
	si.Prompt = te.String(searchNotePromptText).Foreground(common.YellowGreen.Color()).String()
	si.CursorColor = common.Fuschia.String()
	si.CharLimit = noteCharacterLimit
	si.Focus()

	m := stashModel{
		cfg:                cfg,
		authStatus:         as,
		spinner:            sp,
		noteInput:          ni,
		searchInput:        si,
		page:               1,
		paginator:          p,
		loadingFromNetwork: true,
		filesStashing:      make(map[string]struct{}),
	}

	return m
}

// UPDATE

func stashUpdate(msg tea.Msg, m stashModel) (stashModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case errMsg:
		m.err = msg

	case stashLoadErrMsg:
		m.err = msg.err
		m.loaded |= StashedDocuments // still done, albeit unsuccessfully
		m.stashFullyLoaded = true
		m.loadingFromNetwork = false

	case newsLoadErrMsg:
		m.err = msg.err
		m.loaded |= NewsDocuments // still done, albeit unsuccessfully

	// We're finished searching for local files
	case localFileSearchFinished:
		m.loaded |= LocalDocuments

	// Stash results have come in from the server
	case gotStashMsg:
		// This doesn't mean the whole stash listing is loaded, but some we've
		// finished checking for the stash, at least, so mark the stash as
		// loaded here.
		m.loaded |= StashedDocuments

		m.loadingFromNetwork = false

		if len(msg) == 0 {
			// If the server comes back with nothing then we've got everything
			m.stashFullyLoaded = true
		} else {
			docs := wrapMarkdowns(stashedMarkdown, msg)
			m.addMarkdowns(docs...)
		}

	// News has come in from the server
	case gotNewsMsg:
		m.loaded |= NewsDocuments
		if len(msg) > 0 {
			docs := wrapMarkdowns(newsMarkdown, msg)
			m.addMarkdowns(docs...)
		}

	case spinner.TickMsg:
		condition := !m.loadingDone() ||
			m.loadingFromNetwork ||
			m.state == stashStateLoadingDocument ||
			len(m.filesStashing) > 0 ||
			m.spinner.Visible()

		if condition {
			newSpinnerModel, cmd := spinner.Update(msg, m.spinner)
			m.spinner = newSpinnerModel
			cmds = append(cmds, cmd)
		}

	// A note was set on a document. This may have happened in the pager so
	// we'll find the corresponding document here and update accordingly.
	case noteSavedMsg:
		for i := range m.markdowns {
			if m.markdowns[i].ID == msg.ID {
				m.markdowns[i].Note = msg.Note
			}
		}

	// Something was stashed. Add it to the stash listing.
	case stashSuccessMsg:
		md := markdown(msg)
		delete(m.filesStashing, md.localPath) // remove from the things-we're-stashing list

		_ = m.removeLocalMarkdown(md.localPath)
		m.addMarkdowns(&md)

		m.showStatusMessage = true
		m.statusMessage = "Stashed!"
		if m.statusMessageTimer != nil {
			m.statusMessageTimer.Stop()
		}
		m.statusMessageTimer = time.NewTimer(statusMessageTimeout)
		cmds = append(cmds, waitForStatusMessageTimeout(stashContext, m.statusMessageTimer))

	case statusMessageTimeoutMsg:
		if applicationContext(msg) == stashContext {
			m.hideStatusMessage()
		}
	}

	switch m.state {
	case stashStateReady, stashStateShowFiltered:
		pages := len(m.getNotes())

		switch msg := msg.(type) {
		// Handle keys
		case tea.KeyMsg:
			switch msg.String() {
			case "k", "ctrl+k", "up", "shift+tab":
				m.moveCursorUp()

			case "j", "ctrl+j", "down", "tab":
				m.moveCursorDown()

			// Go to the very start
			case "home", "g":
				m.paginator.Page = 0
				m.index = 0

			// Go to the very end
			case "end", "G":
				m.paginator.Page = m.paginator.TotalPages - 1
				m.index = m.paginator.ItemsOnPage(pages) - 1

			// esc only passed trough in stashStateSearchNotes
			case "esc":
				m.state = stashStateReady
				m.searchInput.SetValue("") // clear the searchInput
				m.setTotalPages()

			// Open document
			case "enter":
				m.hideStatusMessage()

				if pages == 0 {
					break
				}

				// Load the document from the server. We'll handle the message
				// that comes back in the main update function.
				md := m.selectedMarkdown()
				m.state = stashStateLoadingDocument

				if md.markdownType == localMarkdown {
					cmds = append(cmds, loadLocalMarkdown(md))
				} else {
					cmds = append(cmds, loadRemoteMarkdown(m.cc, md.ID, md.markdownType))
				}

				cmds = append(cmds, spinner.Tick(m.spinner))

			// Search through your notes
			case "/":
				m.hideStatusMessage()

				m.paginator.Page = 0
				m.index = 0
				m.state = stashStateSearchNotes
				m.searchInput.CursorEnd()
				m.searchInput.Focus()
				return m, textinput.Blink(m.searchInput)

			// Set note
			case "m":
				m.hideStatusMessage()

				if pages == 0 {
					break
				}

				md := m.selectedMarkdown()
				isUserMarkdown := md.markdownType == stashedMarkdown || md.markdownType == convertedMarkdown
				isSettingNote := m.state == stashStateSettingNote
				isPromptingDelete := m.state == stashStatePromptDelete

				if isUserMarkdown && !isSettingNote && !isPromptingDelete {
					m.state = stashStateSettingNote
					m.noteInput.SetValue(md.Note)
					m.noteInput.CursorEnd()
					return m, textinput.Blink(m.noteInput)
				}

			// Stash
			case "s":
				if pages == 0 || m.authStatus != authOK || m.selectedMarkdown() == nil {
					break
				}

				md := m.selectedMarkdown()

				// is the file in the process of being stashed?
				_, isBeingStashed := m.filesStashing[md.localPath]

				isLocalMarkdown := md.markdownType == localMarkdown
				markdownPathMissing := md.localPath == ""

				if isBeingStashed || !isLocalMarkdown || markdownPathMissing {
					if debug && isBeingStashed {
						log.Printf("refusing to stash markdown; we're already stashing %s", md.localPath)
					} else if debug && isLocalMarkdown && markdownPathMissing {
						log.Printf("refusing to stash markdown; local path is empty: %#v", md)
					}
					break
				}

				// Checks passed; perform the stash
				m.filesStashing[md.localPath] = struct{}{}
				cmds = append(cmds, stashDocument(m.cc, *md))

				if m.loadingDone() && !m.spinner.Visible() {
					m.spinner.Start()
					cmds = append(cmds, spinner.Tick(m.spinner))
				}

			// Prompt for deletion
			case "x":
				m.hideStatusMessage()

				if pages == 0 {
					break
				}

				t := m.selectedMarkdown().markdownType
				isUserMarkdown := t == stashedMarkdown || t == convertedMarkdown
				isValidState := m.state != stashStateSettingNote

				if isUserMarkdown && isValidState {
					m.state = stashStatePromptDelete
				}

			// Show errors
			case "!":
				if m.err != nil && m.state == stashStateReady {
					m.state = stashStateShowingError
					return m, nil
				}
			}
		}

		// Update paginator. Pagination key handling is done here, but it could
		// also be moved up to this level, in which case we'd use model methods
		// like model.PageUp().
		newPaginatorModel, cmd := paginator.Update(msg, m.paginator)
		m.paginator = newPaginatorModel
		cmds = append(cmds, cmd)

		// Extra paginator keystrokes
		if key, ok := msg.(tea.KeyMsg); ok {
			if key.Type == tea.KeyRune {
				switch key.Rune {
				case 'b', 'u':
					m.paginator.PrevPage()
				case 'f', 'd':
					m.paginator.NextPage()
				}
			}
		}

		// Keep the index in bounds when paginating
		itemsOnPage := m.paginator.ItemsOnPage(len(m.getNotes()))
		if m.index > itemsOnPage-1 {
			m.index = max(0, itemsOnPage-1)
		}

		// If we're on the last page and we haven't loaded everything, get
		// more stuff.
		if m.paginator.OnLastPage() && !m.loadingFromNetwork && !m.stashFullyLoaded {
			m.page++
			m.loadingFromNetwork = true
			cmds = append(cmds, loadStash(m))
		}

	case stashStatePromptDelete:
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			// Confirm deletion
			case "y":
				if m.state != stashStatePromptDelete {
					break
				}

				smd := m.selectedMarkdown()
				for i, md := range m.markdowns {
					if md != smd {
						continue
					}

					if md.markdownType == convertedMarkdown {
						// If document was stashed during this session, convert it
						// back to a local file.
						md.markdownType = localMarkdown
						md.Note = m.markdowns[i].displayPath
					} else {
						// Delete optimistically and remove the stashed item
						// before we've received a success response.
						m.markdowns = append(m.markdowns[:i], m.markdowns[i+1:]...)
					}
				}

				// Set state and delete
				m.state = stashStateReady
				if m.searchInput.Value() != "" {
					m.state = stashStateShowFiltered
				}

				// Update pagination
				m.setTotalPages()
				return m, deleteStashedItem(m.cc, smd.ID)

			default:
				m.state = stashStateReady
				if m.searchInput.Value() != "" {
					m.state = stashStateShowFiltered
				}
			}
		}

	case stashStateSearchNotes:
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "esc":
				// Cancel search
				m.state = stashStateReady
				m.searchInput.Reset()
			case "enter", "tab", "shift+tab", "ctrl+k", "up", "ctrl+j", "down":
				m.hideStatusMessage()

				if len(m.markdowns) == 0 {
					break
				}

				h := m.getNotes()

				// If we've filtered down to nothing, cancel the search
				if len(h) == 0 {
					m.state = stashStateReady
					m.searchInput.Reset()
					break
				}

				// When there's only one filtered markdown left and the cursor
				// is placed on it we can diretly "enter" it and change to the
				// markdown view
				if len(h) == 1 && h[0].ID == m.selectedMarkdown().ID {
					m.state = stashStateReady
					m.searchInput.Reset()

					var cmd tea.Cmd
					m, cmd = stashUpdate(msg, m)
					cmds = append(cmds, cmd)
					break
				}

				m.searchInput.Blur()

				m.state = stashStateShowFiltered
				if m.searchInput.Value() == "" {
					m.searchInput.Reset()
					m.state = stashStateReady
				}
			}
		}

		// Update the text input component
		newSearchInputModel, cmd := textinput.Update(msg, m.searchInput)
		m.searchInput = newSearchInputModel
		cmds = append(cmds, cmd)

		// Update pagination
		m.setTotalPages()

	case stashStateSettingNote:
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "esc":
				// Cancel note
				m.state = stashStateReady
				if m.searchInput.Value() != "" {
					m.state = stashStateShowFiltered
				}
				m.noteInput.Reset()
			case "enter":
				// Set new note
				md := m.selectedMarkdown()
				newNote := m.noteInput.Value()
				cmd := saveDocumentNote(m.cc, md.ID, newNote)
				md.Note = newNote
				m.noteInput.Reset()
				m.state = stashStateReady
				if m.searchInput.Value() != "" {
					m.state = stashStateShowFiltered
				}
				return m, cmd
			}
		}

		// Update the text input component used to set notes
		newNoteInputModel, cmd := textinput.Update(msg, m.noteInput)
		m.noteInput = newNoteInputModel
		cmds = append(cmds, cmd)

	case stashStateShowingError:
		// Any key exists the error view
		if _, ok := msg.(tea.KeyMsg); ok {
			m.state = stashStateReady
		}
	}

	// If an item is being confirmed for delete, any key (other than the key
	// used for confirmation above) cancels the deletion
	return m, tea.Batch(cmds...)
}

// VIEW

func stashView(m stashModel) string {
	var s string
	switch m.state {
	case stashStateShowingError:
		return errorView(m.err, false)
	case stashStateLoadingDocument:
		s += " " + spinner.View(m.spinner) + " Loading document..."
	case stashStateReady, stashStateSettingNote, stashStatePromptDelete, stashStateSearchNotes, stashStateShowFiltered:

		loadingIndicator := " "
		if !m.localOnly() && (!m.loadingDone() || m.loadingFromNetwork || m.spinner.Visible()) {
			loadingIndicator = spinner.View(m.spinner)
		}

		// We need to fill any empty height with newlines so the footer reaches
		// the bottom.
		numBlankLines := max(0, (m.terminalHeight-stashViewTopPadding-stashViewBottomPadding)%stashViewItemHeight)
		blankLines := ""
		if numBlankLines > 0 {
			blankLines = strings.Repeat("\n", numBlankLines)
		}

		var header string
		if m.showStatusMessage {
			header = greenFg(m.statusMessage)
		} else {
			switch m.state {
			case stashStatePromptDelete:
				header = redFg("Delete this item from your stash? ") + faintRedFg("(y/N)")
			case stashStateSettingNote:
				header = yellowFg("Set the memo for this item?")
			}
		}

		// Only draw the normal header if we're not using the header area for
		// something else (like a prompt or status message)
		if header == "" {
			header = stashHeaderView(m)
		}

		logoOrSearch := glowLogoView(" Glow ")

		// If we're filtering we replace the logo with the search field
		if m.state == stashStateSearchNotes {
			logoOrSearch = textinput.View(m.searchInput)
		} else if m.state == stashStateShowFiltered {
			m.searchInput.TextColor = m.searchInput.PlaceholderColor
			logoOrSearch = textinput.View(m.searchInput)
		}

		var pagination string
		if m.paginator.TotalPages > 1 {
			pagination = paginator.View(m.paginator)

			// If the dot pagination is wider than the width of the window
			// switch to the arabic paginator.
			if ansi.PrintableRuneWidth(pagination) > m.terminalWidth-stashViewHorizontalPadding {
				m.paginator.Type = paginator.Arabic
				pagination = common.Subtle(paginator.View(m.paginator))
			}

			// We could also look at m.stashFullyLoaded and add an indicator
			// showing that we don't actually know how many more pages there
			// are.
		}

		s += fmt.Sprintf(
			"%s %s\n\n  %s\n\n%s\n\n%s  %s\n\n  %s",
			loadingIndicator,
			logoOrSearch,
			header,
			stashPopulatedView(m),
			blankLines,
			pagination,
			stashHelpView(m),
		)
	}
	return "\n" + indent(s, stashIndent)
}

func glowLogoView(text string) string {
	return te.String(text).
		Bold().
		Foreground(glowLogoTextColor).
		Background(common.Fuschia.Color()).
		String()
}

func stashHeaderView(m stashModel) string {
	loading := !m.loadingDone()
	noMarkdowns := len(m.markdowns) == 0

	if m.authStatus == authFailed && m.stashedOnly() {
		return common.Subtle("Can’t load stash. Are you offline?")
	}

	var maybeOffline string
	if m.authStatus == authFailed {
		maybeOffline = " " + offlineHeaderNote
	}

	// Still loading. We haven't found files, stashed items, or news yet.
	if loading && noMarkdowns {
		if m.stashedOnly() {
			return common.Subtle("Loading your stash...")
		} else {
			return common.Subtle("Looking for stuff...") + maybeOffline
		}
	}

	localItems := m.countMarkdowns(localMarkdown)
	stashedItems := m.countMarkdowns(stashedMarkdown) + m.countMarkdowns(convertedMarkdown)
	newsItems := m.countMarkdowns(newsMarkdown)

	// Loading's finished and all we have is news.
	if !loading && localItems == 0 && stashedItems == 0 && newsItems == 0 {
		if m.stashedOnly() {
			return common.Subtle("No stashed markdown files found.") + maybeOffline
		} else {
			return common.Subtle("No local or stashed markdown files found.") + maybeOffline
		}
	}

	// There are local and/or stashed files, so display counts.
	var s string
	if localItems > 0 {
		s += common.Subtle(fmt.Sprintf("%d Local", localItems))
	}
	if stashedItems > 0 {
		var divider string
		if localItems > 0 {
			divider = dividerDot
		}
		si := common.Subtle(fmt.Sprintf("%d Stashed", stashedItems))
		s += fmt.Sprintf("%s%s", divider, si)
	}
	if newsItems > 0 {
		var divider string
		if localItems > 0 || stashedItems > 0 {
			divider = dividerDot
		}
		si := common.Subtle(fmt.Sprintf("%d News", newsItems))

		s += fmt.Sprintf("%s%s", divider, si)
	}
	return common.Subtle(s) + maybeOffline
}

func stashPopulatedView(m stashModel) string {
	var b strings.Builder

	markdowns := m.getNotes()
	if len(markdowns) > 0 {
		start, end := m.paginator.GetSliceBounds(len(markdowns))
		docs := markdowns[start:end]

		for i, md := range docs {
			stashItemView(&b, m, i, md)
			if i != len(docs)-1 {
				fmt.Fprintf(&b, "\n\n")
			}
		}
	}

	// If there aren't enough items to fill up this page (always the last page)
	// then we need to add some newlines to fill up the space where stash items
	// would have been.
	itemsOnPage := m.paginator.ItemsOnPage(len(markdowns))
	if itemsOnPage < m.paginator.PerPage {
		n := (m.paginator.PerPage - itemsOnPage) * stashViewItemHeight
		if len(markdowns) == 0 {
			n -= stashViewItemHeight - 1
		}
		for i := 0; i < n; i++ {
			fmt.Fprint(&b, "\n")
		}
	}

	return b.String()
}

func stashHelpView(m stashModel) string {
	var (
		h         []string
		isStashed bool
		isLocal   bool
		numDocs   = len(m.getNotes())
	)

	if numDocs > 0 {
		md := m.selectedMarkdown()
		isStashed = md != nil && md.markdownType == stashedMarkdown
		isLocal = md != nil && md.markdownType == localMarkdown
	}

	if m.state == stashStateSettingNote {
		h = append(h, "enter: confirm", "esc: cancel")
	} else if m.state == stashStatePromptDelete {
		h = append(h, "y: delete", "n: cancel")
	} else if m.state == stashStateSearchNotes && numDocs == 1 {
		h = append(h, "enter: open", "esc: cancel")
	} else if m.state == stashStateSearchNotes && numDocs == 0 {
		h = append(h, "enter/esc: cancel")
	} else if m.state == stashStateSearchNotes {
		h = append(h, "enter: confirm", "esc: cancel", "ctrl+j/ctrl+k, ↑/↓: choose")
	} else {
		if len(m.markdowns) > 0 {
			h = append(h, "enter: open")
		}
		if m.state == stashStateShowFiltered {
			h = append(h, "esc: clear search")
		}
		if len(m.markdowns) > 1 {
			h = append(h, "j/k, ↑/↓: choose")
		}
		if m.paginator.TotalPages > 1 {
			h = append(h, "h/l, ←/→: page")
		}
		if isStashed {
			h = append(h, "x: delete", "m: set memo")
		} else if isLocal && m.authStatus == authOK {
			h = append(h, "s: stash")
		}
		if m.err != nil {
			h = append(h, "!: errors")
		}
		h = append(h, "/: search")
		h = append(h, "q: quit")
	}
	return stashHelpViewBuilder(m.terminalWidth, h...)
}

// builds the help view from various sections pieces, truncating it if the view
// would otherwise wrap to two lines.
func stashHelpViewBuilder(windowWidth int, sections ...string) string {
	if len(sections) == 0 {
		return ""
	}

	const truncationWidth = 1 // width of "…"

	var (
		s        string
		next     string
		maxWidth = windowWidth - stashViewHorizontalPadding - truncationWidth
	)

	for i := 0; i < len(sections); i++ {
		// If we need this more often we'll formalize something rather than
		// use an if clause/switch here.
		switch sections[i] {
		case "s: stash":
			next = greenFg(sections[i])
		default:
			next = stashHelpItemStyle(sections[i])
		}

		if i < len(sections)-1 {
			next += dividerDot
		}

		// Only this (and the following) help text items if we have the
		// horizontal space
		if ansi.PrintableRuneWidth(s)+ansi.PrintableRuneWidth(next) >= maxWidth {
			s += common.Subtle("…")
			break
		}

		s += next
	}
	return s
}

// COMMANDS

func loadRemoteMarkdown(cc *charm.Client, id int, t markdownType) tea.Cmd {
	return func() tea.Msg {
		var (
			md  *charm.Markdown
			err error
		)

		if t == stashedMarkdown || t == convertedMarkdown {
			md, err = cc.GetStashMarkdown(id)
		} else {
			md, err = cc.GetNewsMarkdown(id)
		}

		if err != nil {
			if debug {
				log.Println("error loading remote markdown:", err)
			}
			return errMsg{err}
		}

		return fetchedMarkdownMsg(&markdown{
			markdownType: t,
			Markdown:     *md,
		})
	}
}

func loadLocalMarkdown(md *markdown) tea.Cmd {
	return func() tea.Msg {
		if md.markdownType != localMarkdown {
			return errMsg{errors.New("could not load local file: not a local file")}
		}
		if md.localPath == "" {
			return errMsg{errors.New("could not load file: missing path")}
		}

		data, err := ioutil.ReadFile(md.localPath)
		if err != nil {
			if debug {
				log.Println("error reading local markdown:", err)
			}
			return errMsg{err}
		}
		md.Body = string(data)
		return fetchedMarkdownMsg(md)
	}
}

func deleteStashedItem(cc *charm.Client, id int) tea.Cmd {
	return func() tea.Msg {
		err := cc.DeleteMarkdown(id)
		if err != nil {
			if debug {
				log.Println("could not delete stashed item:", err)
			}
			return errMsg{err}
		}
		return deletedStashedItemMsg(id)
	}
}

// ETC

// wrapMarkdowns wraps a *charm.Markdown with a *markdown in order to add some
// extra metadata.
func wrapMarkdowns(t markdownType, md []*charm.Markdown) (m []*markdown) {
	for _, v := range md {
		m = append(m, &markdown{
			markdownType: t,
			Markdown:     *v,
		})
	}
	return m
}

func truncate(str string, num int) string {
	return runewidth.Truncate(str, num, "…")
}

var magnitudes = []humanize.RelTimeMagnitude{
	{D: time.Second, Format: "now", DivBy: time.Second},
	{D: 2 * time.Second, Format: "1 second %s", DivBy: 1},
	{D: time.Minute, Format: "%d seconds %s", DivBy: time.Second},
	{D: 2 * time.Minute, Format: "1 minute %s", DivBy: 1},
	{D: time.Hour, Format: "%d minutes %s", DivBy: time.Minute},
	{D: 2 * time.Hour, Format: "1 hour %s", DivBy: 1},
	{D: humanize.Day, Format: "%d hours %s", DivBy: time.Hour},
	{D: 2 * humanize.Day, Format: "1 day %s", DivBy: 1},
	{D: humanize.Week, Format: "%d days %s", DivBy: humanize.Day},
	{D: 2 * humanize.Week, Format: "1 week %s", DivBy: 1},
	{D: humanize.Month, Format: "%d weeks %s", DivBy: humanize.Week},
	{D: 2 * humanize.Month, Format: "1 month %s", DivBy: 1},
	{D: humanize.Year, Format: "%d months %s", DivBy: humanize.Month},
	{D: 18 * humanize.Month, Format: "1 year %s", DivBy: 1},
	{D: 2 * humanize.Year, Format: "2 years %s", DivBy: 1},
	{D: humanize.LongTime, Format: "%d years %s", DivBy: humanize.Year},
	{D: math.MaxInt64, Format: "a long while %s", DivBy: 1},
}

func relativeTime(then time.Time) string {
	now := time.Now()
	ago := now.Sub(then)
	if ago < time.Minute {
		return "just now"
	} else if ago < humanize.Week {
		return humanize.CustomRelTime(then, now, "ago", "from now", magnitudes)
	}
	return then.Format("02 Jan 2006 15:04 MST")
}
