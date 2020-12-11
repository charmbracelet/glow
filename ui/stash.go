package ui

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm"
	lib "github.com/charmbracelet/charm/ui/common"
	"github.com/muesli/reflow/ansi"
	te "github.com/muesli/termenv"
	"github.com/sahilm/fuzzy"
)

const (
	stashIndent                = 1
	stashViewItemHeight        = 3 // height of stash entry, including gap
	stashViewTopPadding        = 5 // logo, status bar, gaps
	stashViewBottomPadding     = 3 // pagination and gaps, but not help
	stashViewHorizontalPadding = 6
)

var (
	stashTextInputPromptStyle styleFunc = newFgStyle(lib.YellowGreen)
	dividerDot                string    = darkGrayFg(" • ")
	dividerBar                string    = darkGrayFg(" │ ")
	offlineHeaderNote         string    = darkGrayFg("(Offline)")
)

// MSG

type deletedStashedItemMsg int
type filteredMarkdownMsg []*markdown
type fetchedMarkdownMsg *markdown

type markdownFetchFailedMsg struct {
	err  error
	id   int
	note string
}

// MODEL

// stashViewState is the high-level state of the file listing.
type stashViewState int

const (
	stashStateReady stashViewState = iota
	stashStateLoadingDocument
	stashStateShowingError
)

// The types of documents we are currently showing to the user.
type sectionKey int

const (
	localSection = iota
	stashedSection
	newsSection
)

// section contains definitions and state information for displaying a tab and
// its contents in the file listing view.
type section struct {
	key       sectionKey
	docTypes  DocTypeSet
	paginator paginator.Model
	cursor    int
}

// map sections to their associated types.
var sections = map[sectionKey]section{
	localSection: {
		key:      localSection,
		docTypes: NewDocTypeSet(LocalDoc),
	},
	stashedSection: {
		key:      stashedSection,
		docTypes: NewDocTypeSet(StashedDoc, ConvertedDoc),
	},
	newsSection: {
		key:      newsSection,
		docTypes: NewDocTypeSet(NewsDoc),
	},
}

// filterState is the current filtering state in the file listing.
type filterState int

const (
	unfiltered    filterState = iota // no filter set
	filtering                        // user is actively setting a filter
	filterApplied                    // a filter is applied and user is not editing filter
)

// selectionState is the state of the currently selected document.
type selectionState int

const (
	selectionIdle = iota
	selectionSettingNote
	selectionPromptingDelete
)

type stashModel struct {
	common             *commonModel
	err                error
	spinner            spinner.Model
	noteInput          textinput.Model
	filterInput        textinput.Model
	stashFullyLoaded   bool // have we loaded all available stashed documents from the server?
	viewState          stashViewState
	filterState        filterState
	selectionState     selectionState
	showFullHelp       bool
	showStatusMessage  bool
	statusMessage      string
	statusMessageTimer *time.Timer

	// Available document sections we can cycle through. We use a slice, rather
	// than a map, because order is important.
	sections []section

	// Index of the section we're currently looking at
	sectionIndex int

	// Tracks what exactly is loaded between the stash, news and local files
	loaded DocTypeSet

	// The master set of markdown documents we're working with.
	markdowns []*markdown

	// Markdown documents we're currently displaying. Filtering, toggles and so
	// on will alter this slice so we can show what is relevant. For that
	// reason, this field should be considered ephemeral.
	filteredMarkdowns []*markdown

	// Page we're fetching stash items from on the server, which is different
	// from the local pagination. Generally, the server will return more items
	// than we can display at a time so we can paginate locally without having
	// to fetch every time.
	serverPage int
}

func (m stashModel) localOnly() bool {
	return m.common.cfg.localOnly()
}

func (m stashModel) stashedOnly() bool {
	return m.common.cfg.stashedOnly()
}

func (m stashModel) loadingDone() bool {
	return m.loaded.Equals(m.common.cfg.DocumentTypes.Difference(ConvertedDoc))
}

func (m stashModel) hasSection(key sectionKey) bool {
	for _, v := range m.sections {
		if key == v.key {
			return true
		}
	}
	return false
}

func (m stashModel) currentSection() *section {
	return &m.sections[m.sectionIndex]
}

func (m stashModel) paginator() *paginator.Model {
	return &m.currentSection().paginator
}

func (m *stashModel) setPaginator(p paginator.Model) {
	m.currentSection().paginator = p
}

func (m stashModel) cursor() int {
	return m.currentSection().cursor
}

func (m *stashModel) setCursor(i int) {
	m.currentSection().cursor = i
}

// Returns whether or not we're online. That is, when "local-only" mode is
// disabled and we've authenticated successfully.
func (m stashModel) online() bool {
	return !m.localOnly() && m.common.authStatus == authOK
}

func (m *stashModel) setSize(width, height int) {
	m.common.width = width
	m.common.height = height

	m.noteInput.Width = width - stashViewHorizontalPadding*2 - ansi.PrintableRuneWidth(m.noteInput.Prompt)
	m.filterInput.Width = width - stashViewHorizontalPadding*2 - ansi.PrintableRuneWidth(m.filterInput.Prompt)

	m.updatePagination()
}

func (m *stashModel) resetFiltering() {
	m.filterState = unfiltered
	m.filterInput.Reset()
	sort.Stable(markdownsByLocalFirst(m.markdowns))
	m.filteredMarkdowns = nil
	m.updatePagination()
}

// Is a filter currently being applied?
func (m stashModel) isFiltering() bool {
	return m.filterState != unfiltered
}

// Should we be updating the filter?
func (m stashModel) shouldUpdateFilter() bool {
	// If we're in the middle of setting a note don't update the filter so that
	// the focus won't jump around.
	return m.isFiltering() && m.selectionState != selectionSettingNote
}

// Update pagination according to the amount of markdowns for the current
// state.
func (m *stashModel) updatePagination() {
	_, helpHeight := m.helpView()

	availableHeight := m.common.height -
		stashViewTopPadding -
		helpHeight -
		stashViewBottomPadding

	m.paginator().PerPage = max(1, availableHeight/stashViewItemHeight)

	if pages := len(m.getVisibleMarkdowns()); pages < 1 {
		m.paginator().SetTotalPages(1)
	} else {
		m.paginator().SetTotalPages(pages)
	}

	// Make sure the page stays in bounds
	if m.paginator().Page >= m.paginator().TotalPages-1 {
		m.paginator().Page = max(0, m.paginator().TotalPages-1)
	}
}

// MarkdownIndex returns the index of the currently selected markdown item.
func (m stashModel) markdownIndex() int {
	return m.paginator().Page*m.paginator().PerPage + m.cursor()
}

// Return the current selected markdown in the stash.
func (m stashModel) selectedMarkdown() *markdown {
	i := m.markdownIndex()

	mds := m.getVisibleMarkdowns()
	if i < 0 || len(mds) == 0 || len(mds) <= i {
		return nil
	}

	return mds[i]
}

// Adds markdown documents to the model.
func (m *stashModel) addMarkdowns(mds ...*markdown) {
	if len(mds) > 0 {
		for _, md := range mds {
			md.generateLocalID()
		}

		m.markdowns = append(m.markdowns, mds...)
		if !m.isFiltering() {
			sort.Stable(markdownsByLocalFirst(m.markdowns))
		}
		m.updatePagination()
	}
}

// Return the number of markdown documents of a given type.
func (m stashModel) countMarkdowns(t DocType) (found int) {
	if len(m.markdowns) == 0 {
		return
	}

	var mds []*markdown
	if m.isFiltering() {
		mds = m.getVisibleMarkdowns()
	} else {
		mds = m.markdowns
	}

	for i := 0; i < len(mds); i++ {
		if mds[i].markdownType == t {
			found++
		}
	}
	return
}

// Sift through the master markdown collection for the specified types.
func (m stashModel) getMarkdownByType(types ...DocType) []*markdown {
	var agg []*markdown

	if len(m.markdowns) == 0 {
		return agg
	}

	for _, t := range types {
		for _, md := range m.markdowns {
			if md.markdownType == t {
				agg = append(agg, md)
			}
		}
	}

	sort.Sort(markdownsByLocalFirst(agg))
	return agg
}

// Returns the markdowns that should be currently shown.
func (m stashModel) getVisibleMarkdowns() []*markdown {
	if m.isFiltering() {
		return m.filteredMarkdowns
	}

	return m.getMarkdownByType(m.currentSection().docTypes.AsSlice()...)
}

// Return the markdowns eligible to be filtered.
func (m stashModel) getFilterableMarkdowns() []*markdown {
	return m.getMarkdownByType(LocalDoc, ConvertedDoc, StashedDoc)
}

// Command for opening a markdown document in the pager. Note that this also
// alters the model.
func (m *stashModel) openMarkdown(md *markdown) tea.Cmd {
	var cmd tea.Cmd
	m.viewState = stashStateLoadingDocument

	if md.markdownType == LocalDoc {
		cmd = loadLocalMarkdown(md)
	} else {
		cmd = loadRemoteMarkdown(m.common.cc, md)
	}

	return tea.Batch(cmd, spinner.Tick)
}

func (m *stashModel) newStatusMessage(s string) tea.Cmd {
	m.showStatusMessage = true
	m.statusMessage = s
	if m.statusMessageTimer != nil {
		m.statusMessageTimer.Stop()
	}
	m.statusMessageTimer = time.NewTimer(statusMessageTimeout)
	return waitForStatusMessageTimeout(stashContext, m.statusMessageTimer)
}

func (m *stashModel) hideStatusMessage() {
	m.showStatusMessage = false
	m.statusMessage = ""
	if m.statusMessageTimer != nil {
		m.statusMessageTimer.Stop()
	}
}

func (m *stashModel) moveCursorUp() {
	m.setCursor(m.cursor() - 1)
	if m.cursor() < 0 && m.paginator().Page == 0 {
		// Stop
		m.setCursor(0)
		return
	}

	if m.cursor() >= 0 {
		return
	}
	// Go to previous page
	m.paginator().PrevPage()

	m.setCursor(m.paginator().ItemsOnPage(len(m.getVisibleMarkdowns())) - 1)
}

func (m *stashModel) moveCursorDown() {
	itemsOnPage := m.paginator().ItemsOnPage(len(m.getVisibleMarkdowns()))

	m.setCursor(m.cursor() + 1)
	if m.cursor() < itemsOnPage {
		return
	}

	if !m.paginator().OnLastPage() {
		m.paginator().NextPage()
		m.setCursor(0)
		return
	}

	// During filtering the cursor position can exceed the number of
	// itemsOnPage. It's more intuitive to start the cursor at the
	// topmost position when moving it down in this scenario.
	if m.cursor() > itemsOnPage {
		m.setCursor(0)
		return
	}
	m.setCursor(itemsOnPage - 1)
}

// INIT

func newStashModel(common *commonModel) stashModel {
	sp := spinner.NewModel()
	sp.Spinner = spinner.Line
	sp.ForegroundColor = lib.SpinnerColor.String()
	sp.HideFor = time.Millisecond * 50
	sp.MinimumLifetime = time.Millisecond * 180
	sp.Start()

	ni := textinput.NewModel()
	ni.Prompt = stashTextInputPromptStyle("Memo: ")
	ni.CursorColor = lib.Fuschia.String()
	ni.CharLimit = noteCharacterLimit
	ni.Focus()

	si := textinput.NewModel()
	si.Prompt = stashTextInputPromptStyle("Filter: ")
	si.CursorColor = lib.Fuschia.String()
	si.CharLimit = noteCharacterLimit
	si.Focus()

	var s []section
	if common.cfg.localOnly() {
		s = []section{
			sections[localSection],
		}
	} else if common.cfg.stashedOnly() {
		s = []section{
			sections[stashedSection],
			sections[newsSection],
		}
	} else {
		s = []section{
			sections[localSection],
			sections[stashedSection],
			sections[newsSection],
		}
	}

	p := paginator.NewModel()
	p.Type = paginator.Dots
	p.ActiveDot = brightGrayFg("•")
	p.InactiveDot = darkGrayFg("•")

	for i := range s {
		s[i].paginator = p
	}

	m := stashModel{
		common:      common,
		spinner:     sp,
		noteInput:   ni,
		filterInput: si,
		serverPage:  1,
		loaded:      NewDocTypeSet(),
		sections:    s,
	}

	return m
}

// UPDATE

func (m stashModel) update(msg tea.Msg) (stashModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case errMsg:
		m.err = msg

	case stashLoadErrMsg:
		m.err = msg.err
		m.loaded.Add(StashedDoc) // still done, albeit unsuccessfully
		m.stashFullyLoaded = true

	case newsLoadErrMsg:
		m.err = msg.err
		m.loaded.Add(NewsDoc) // still done, albeit unsuccessfully

	case localFileSearchFinished:
		// We're finished searching for local files
		m.loaded.Add(LocalDoc)

	case gotStashMsg, gotNewsMsg:
		// Stash or news results have come in from the server.
		//
		// With the stash, this doesn't mean the whole stash listing is loaded,
		// but now know it can load, at least, so mark the stash as loaded here.
		var docs []*markdown

		switch msg := msg.(type) {
		case gotStashMsg:
			m.loaded.Add(StashedDoc)
			docs = wrapMarkdowns(StashedDoc, msg)

			if len(msg) == 0 {
				// If the server comes back with nothing then we've got
				// everything
				m.stashFullyLoaded = true
			} else {
				// Load the next page
				m.serverPage++
				cmds = append(cmds, loadStash(m))
			}

		case gotNewsMsg:
			m.loaded.Add(NewsDoc)
			docs = wrapMarkdowns(NewsDoc, msg)
		}

		// If we're filtering build filter indexes immediately so any
		// matching results will show up in the filter.
		if m.isFiltering() {
			for _, md := range docs {
				md.buildFilterValue()
			}
		}
		if m.shouldUpdateFilter() {
			cmds = append(cmds, filterMarkdowns(m))
		}

		m.addMarkdowns(docs...)

	case markdownFetchFailedMsg:
		s := "Couldn't load markdown"
		if msg.note != "" {
			s += ": " + msg.note
		}
		cmd := m.newStatusMessage(s)
		return m, cmd

	case filteredMarkdownMsg:
		m.filteredMarkdowns = msg
		return m, nil

	case spinner.TickMsg:
		loading := !m.loadingDone()
		stashing := m.common.isStashing()
		openingDocument := m.viewState == stashStateLoadingDocument
		spinnerVisible := m.spinner.Visible()

		if loading || stashing || openingDocument || spinnerVisible {
			newSpinnerModel, cmd := m.spinner.Update(msg)
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

	// Note: mechanical stuff related to stash success is handled in the parent
	// update function.
	case stashSuccessMsg:
		cmds = append(cmds, m.newStatusMessage("Stashed!"))

	// Note: mechanical stuff related to stash failure is handled in the parent
	// update function.
	case stashFailMsg:
		cmds = append(cmds, m.newStatusMessage("Couldn’t stash :("))

	case statusMessageTimeoutMsg:
		if applicationContext(msg) == stashContext {
			m.hideStatusMessage()
		}
	}

	if m.filterState == filtering {
		cmds = append(cmds, m.handleFiltering(msg))
		return m, tea.Batch(cmds...)
	}

	switch m.selectionState {
	case selectionSettingNote:
		cmds = append(cmds, m.handleNoteInput(msg))
		return m, tea.Batch(cmds...)
	case selectionPromptingDelete:
		cmds = append(cmds, m.handleDeleteConfirmation(msg))
		return m, tea.Batch(cmds...)
	}

	// Updates per the current state
	switch m.viewState {
	case stashStateReady:
		cmds = append(cmds, m.handleDocumentBrowsing(msg))
	case stashStateShowingError:
		// Any key exists the error view
		if _, ok := msg.(tea.KeyMsg); ok {
			m.viewState = stashStateReady
		}
	}

	return m, tea.Batch(cmds...)
}

// Updates for when a user is browsing the markdown listing.
func (m *stashModel) handleDocumentBrowsing(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	numDocs := len(m.getVisibleMarkdowns())

	switch msg := msg.(type) {
	// Handle keys
	case tea.KeyMsg:
		switch msg.String() {
		case "k", "ctrl+k", "up":
			m.moveCursorUp()

		case "j", "ctrl+j", "down":
			m.moveCursorDown()

		// Go to the very start
		case "home", "g":
			m.paginator().Page = 0
			m.setCursor(0)

		// Go to the very end
		case "end", "G":
			m.paginator().Page = m.paginator().TotalPages - 1
			m.setCursor(m.paginator().ItemsOnPage(numDocs) - 1)

		case "esc":
			if m.isFiltering() {
				m.resetFiltering()
				break
			}

		case "tab":
			if len(m.sections) == 0 || m.isFiltering() {
				break
			}
			m.sectionIndex++
			if m.sectionIndex >= len(m.sections) {
				m.sectionIndex = 0
			}
			m.updatePagination()

		case "shift+tab":
			if len(m.sections) == 0 {
				break
			}
			m.sectionIndex--
			if m.sectionIndex < 0 {
				m.sectionIndex = len(m.sections) - 1
			}
			m.updatePagination()

		// Open document
		case "enter":
			m.hideStatusMessage()

			if numDocs == 0 {
				break
			}

			// Load the document from the server. We'll handle the message
			// that comes back in the main update function.
			md := m.selectedMarkdown()
			cmds = append(cmds, m.openMarkdown(md))

		// Filter your notes
		case "/":
			m.hideStatusMessage()

			// Build values we'll filter against
			for _, md := range m.markdowns {
				md.buildFilterValue()
			}

			m.filteredMarkdowns = m.getFilterableMarkdowns()

			m.paginator().Page = 0
			m.setCursor(0)
			m.filterState = filtering
			m.filterInput.CursorEnd()
			m.filterInput.Focus()
			return textinput.Blink

		// Set note
		case "m":
			m.hideStatusMessage()

			if numDocs == 0 {
				break
			}

			md := m.selectedMarkdown()
			isUserMarkdown := md.markdownType == StashedDoc || md.markdownType == ConvertedDoc
			isSettingNote := m.selectionState == selectionSettingNote
			isPromptingDelete := m.selectionState == selectionPromptingDelete

			if isUserMarkdown && !isSettingNote && !isPromptingDelete {
				m.selectionState = selectionSettingNote
				m.noteInput.SetValue(md.Note)
				m.noteInput.CursorEnd()
				return textinput.Blink
			}

		// Stash
		case "s":
			if numDocs == 0 || !m.online() || m.selectedMarkdown() == nil {
				break
			}

			md := m.selectedMarkdown()

			if !stashableDocTypes.Contains(md.markdownType) {
				break
			}

			if _, alreadyStashed := m.common.filesStashed[md.localID]; alreadyStashed {
				cmds = append(cmds, m.newStatusMessage("Already stashed"))
				break
			}

			if !stashableDocTypes.Contains(md.markdownType) || md.localID.IsNil() {
				if debug && md.localID.IsNil() {
					log.Printf("refusing to stash markdown; local ID path is nil: %#v", md)
				}
				break
			}

			// Checks passed; perform the stash
			m.common.filesStashed[md.localID] = struct{}{}
			m.common.filesStashing[md.localID] = struct{}{}
			cmds = append(cmds, stashDocument(m.common.cc, *md))

			if m.loadingDone() && !m.spinner.Visible() {
				m.spinner.Start()
				cmds = append(cmds, spinner.Tick)
			}

		// Prompt for deletion
		case "x":
			m.hideStatusMessage()

			validState := m.viewState == stashStateReady &&
				m.selectionState == selectionIdle

			if numDocs == 0 && !validState {
				break
			}

			md := m.selectedMarkdown()
			if md == nil {
				break
			}

			t := md.markdownType
			if t == StashedDoc || t == ConvertedDoc {
				m.selectionState = selectionPromptingDelete
			}

		// Toggle full help
		case "?":
			m.showFullHelp = !m.showFullHelp
			m.updatePagination()

		// Show errors
		case "!":
			if m.err != nil && m.viewState == stashStateReady {
				m.viewState = stashStateShowingError
				return nil
			}
		}
	}

	// Update paginator. Pagination key handling is done here, but it could
	// also be moved up to this level, in which case we'd use model methods
	// like model.PageUp().
	newPaginatorModel, cmd := m.paginator().Update(msg)
	m.setPaginator(newPaginatorModel)
	cmds = append(cmds, cmd)

	// Extra paginator keystrokes
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "b", "u":
			m.paginator().PrevPage()
		case "f", "d":
			m.paginator().NextPage()
		}
	}

	// Keep the index in bounds when paginating
	itemsOnPage := m.paginator().ItemsOnPage(len(m.getVisibleMarkdowns()))
	if m.cursor() > itemsOnPage-1 {
		m.setCursor(max(0, itemsOnPage-1))
	}

	return tea.Batch(cmds...)
}

// Updates for when a user is being prompted whether or not to delete a
// markdown item.
func (m *stashModel) handleDeleteConfirmation(msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "y":
			if m.selectionState != selectionPromptingDelete {
				break
			}

			smd := m.selectedMarkdown()
			for i, md := range m.markdowns {
				if md != smd {
					continue
				}

				// Remove from the things-we-stashed-this-session set
				delete(m.common.filesStashed, md.localID)

				// Delete optimistically and remove the stashed item before
				// we've received a success response.
				if m.isFiltering() {
					mds, _ := deleteMarkdown(m.filteredMarkdowns, m.markdowns[i])
					m.filteredMarkdowns = mds
				}
				mds, _ := deleteMarkdown(m.markdowns, m.markdowns[i])
				m.markdowns = mds
			}

			m.selectionState = selectionIdle
			m.updatePagination()

			return deleteStashedItem(m.common.cc, smd.ID)

		default:
			// Any other keys cancels deletion
			m.selectionState = selectionIdle
		}
	}

	return nil
}

// Updates for when a user is in the filter editing interface.
func (m *stashModel) handleFiltering(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	// Handle keys
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc":
			// Cancel filtering
			m.resetFiltering()
		case "enter", "tab", "shift+tab", "ctrl+k", "up", "ctrl+j", "down":
			m.hideStatusMessage()

			if len(m.markdowns) == 0 {
				break
			}

			h := m.getVisibleMarkdowns()

			// If we've filtered down to nothing, clear the filter
			if len(h) == 0 {
				m.viewState = stashStateReady
				m.resetFiltering()
				break
			}

			// When there's only one filtered markdown left we can just
			// "open" it directly
			if len(h) == 1 {
				m.viewState = stashStateReady
				m.resetFiltering()
				cmds = append(cmds, m.openMarkdown(h[0]))
				break
			}

			m.filterInput.Blur()

			m.filterState = filterApplied
			if m.filterInput.Value() == "" {
				m.resetFiltering()
			}
		}
	}

	// Update the filter text input component
	newFilterInputModel, inputCmd := m.filterInput.Update(msg)
	currentFilterVal := m.filterInput.Value()
	newFilterVal := newFilterInputModel.Value()
	m.filterInput = newFilterInputModel
	cmds = append(cmds, inputCmd)

	// If the filtering input has changed, request updated filtering
	if newFilterVal != currentFilterVal {
		cmds = append(cmds, filterMarkdowns(*m))
	}

	// Update pagination
	m.updatePagination()

	return tea.Batch(cmds...)
}

func (m *stashModel) handleNoteInput(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc":
			// Cancel note
			m.noteInput.Reset()
			m.selectionState = selectionIdle
		case "enter":
			// Set new note
			md := m.selectedMarkdown()
			newNote := m.noteInput.Value()
			cmd := saveDocumentNote(m.common.cc, md.ID, newNote)
			md.Note = newNote
			m.noteInput.Reset()
			m.selectionState = selectionIdle
			return cmd
		}
	}

	if m.shouldUpdateFilter() {
		cmds = append(cmds, filterMarkdowns(*m))
	}

	// Update the note text input component
	newNoteInputModel, noteInputCmd := m.noteInput.Update(msg)
	m.noteInput = newNoteInputModel
	cmds = append(cmds, noteInputCmd)

	return tea.Batch(cmds...)
}

// VIEW

func (m stashModel) view() string {
	var s string
	switch m.viewState {
	case stashStateShowingError:
		return errorView(m.err, false)
	case stashStateLoadingDocument:
		s += " " + m.spinner.View() + " Loading document..."
	case stashStateReady:

		loadingIndicator := " "
		if !m.localOnly() && (!m.loadingDone() || m.spinner.Visible()) {
			loadingIndicator = m.spinner.View()
		}

		var header string
		switch m.selectionState {
		case selectionPromptingDelete:
			header = redFg("Delete this item from your stash? ") + faintRedFg("(y/N)")
		case selectionSettingNote:
			header = yellowFg("Set the memo for this item?")
		}

		// Only draw the normal header if we're not using the header area for
		// something else (like a note or delete prompt).
		if header == "" {
			header = m.headerView()
		}

		// Rules for the logo, filter and status message.
		var logoOrFilter string
		if m.showStatusMessage {
			const gutter = 3
			logoOrFilter = greenFg(truncate(m.statusMessage, m.common.width-gutter))
		} else if m.isFiltering() {
			logoOrFilter = m.filterInput.View()
		} else {
			logoOrFilter = glowLogoView(" Glow ")
		}

		help, helpHeight := m.helpView()

		populatedView := m.populatedView()
		populatedViewHeight := strings.Count(populatedView, "\n") + 2

		// We need to fill any empty height with newlines so the footer reaches
		// the bottom.
		availHeight := m.common.height -
			stashViewTopPadding -
			populatedViewHeight -
			helpHeight -
			stashViewBottomPadding
		blankLines := strings.Repeat("\n", max(0, availHeight))

		var pagination string
		if m.paginator().TotalPages > 1 {
			pagination = m.paginator().View()

			// If the dot pagination is wider than the width of the window
			// switch to the arabic paginator.
			if ansi.PrintableRuneWidth(pagination) > m.common.width-stashViewHorizontalPadding {
				m.paginator().Type = paginator.Arabic
				pagination = lib.Subtle(m.paginator().View())
			}

			// We could also look at m.stashFullyLoaded and add an indicator
			// showing that we don't actually know how many more pages there
			// are.
		}

		s += fmt.Sprintf(
			"%s %s\n\n  %s\n\n%s\n\n%s  %s\n\n%s",
			loadingIndicator,
			logoOrFilter,
			header,
			populatedView,
			blankLines,
			pagination,
			help,
		)
	}
	return "\n" + indent(s, stashIndent)
}

func glowLogoView(text string) string {
	return te.String(text).
		Bold().
		Foreground(glowLogoTextColor).
		Background(lib.Fuschia.Color()).
		String()
}

func (m stashModel) headerView() string {
	localCount := m.countMarkdowns(LocalDoc)
	stashedCount := m.countMarkdowns(StashedDoc) + m.countMarkdowns(ConvertedDoc)
	newsCount := m.countMarkdowns(NewsDoc)

	var sections []string

	// Filter results
	if m.isFiltering() {
		if localCount+stashedCount+newsCount == 0 {
			return grayFg("Nothing found.")
		} else {
			if localCount > 0 {
				sections = append(sections, fmt.Sprintf("%d local", localCount))
			}
			if stashedCount > 0 {
				sections = append(sections, fmt.Sprintf("%d stashed", stashedCount))
			}
		}

		for i := range sections {
			sections[i] = grayFg(sections[i])
		}

		return strings.Join(sections, dividerDot)
	}

	if m.loadingDone() && len(m.markdowns) == 0 {
		var maybeOffline string
		if m.common.authStatus == authFailed {
			maybeOffline = " " + offlineHeaderNote
		}

		if m.stashedOnly() {
			return lib.Subtle("Can’t load stash") + maybeOffline
		}
		return lib.Subtle("No markdown files found") + maybeOffline
	}

	// Tabs
	for i, v := range m.sections {
		var s string

		switch v.key {
		case localSection:
			if m.stashedOnly() {
				continue
			}
			s = fmt.Sprintf("%d local", localCount)
		case stashedSection:
			if m.localOnly() {
				continue
			}
			s = fmt.Sprintf("%d stashed", stashedCount)
		case newsSection:
			if m.localOnly() {
				continue
			}
			s = fmt.Sprintf("%d news", newsCount)
		}

		if m.sectionIndex == i && len(m.sections) > 1 {
			s = brightGrayFg(s)
		} else {
			s = grayFg(s)
		}
		sections = append(sections, s)
	}

	s := strings.Join(sections, dividerBar)
	if m.common.authStatus == authFailed {
		s += dividerDot + offlineHeaderNote
	}

	return s
}

func (m stashModel) populatedView() string {
	mds := m.getVisibleMarkdowns()

	if len(mds) == 0 && m.isFiltering() {
		return ""
	}

	var b strings.Builder

	// Empty states
	if len(mds) == 0 {
		f := func(s string) {
			b.WriteString("  " + grayFg(s))
		}

		switch m.sections[m.sectionIndex].key {
		case localSection:
			if m.loadingDone() {
				f("No local files found.")
			} else {
				f("Looking for local files...")
			}
		case stashedSection:
			if m.common.authStatus == authFailed {
				f("Can't load your stash. Are you offline?")
			} else if m.loadingDone() {
				f("Nothing stashed yet.")
			} else {
				f("Loading your stash...")
			}
		case newsSection:
			if m.common.authStatus == authFailed {
				f("Can't load news. Are you offline?")
			} else if m.loadingDone() {
				f("No stashed files found.")
			} else {
				f("Loading your stash...")
			}
		}
	}

	if len(mds) > 0 {
		start, end := m.paginator().GetSliceBounds(len(mds))
		docs := mds[start:end]

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
	itemsOnPage := m.paginator().ItemsOnPage(len(mds))
	if itemsOnPage < m.paginator().PerPage {
		n := (m.paginator().PerPage - itemsOnPage) * stashViewItemHeight
		if len(mds) == 0 {
			n -= stashViewItemHeight - 1
		}
		for i := 0; i < n; i++ {
			fmt.Fprint(&b, "\n")
		}
	}

	return b.String()
}

// COMMANDS

// loadRemoteMarkdown is a command for loading markdown from the server.
func loadRemoteMarkdown(cc *charm.Client, md *markdown) tea.Cmd {
	return func() tea.Msg {
		newMD, err := fetchMarkdown(cc, md.ID, md.markdownType)
		if err != nil {
			if debug {
				log.Printf("error loading %s markdown (ID %d, Note: '%s'): %v", md.markdownType, md.ID, md.Note, err)
			}
			return markdownFetchFailedMsg{
				err:  err,
				id:   md.ID,
				note: md.Note,
			}
		}
		newMD.localID = md.localID
		return fetchedMarkdownMsg(newMD)
	}
}

func loadLocalMarkdown(md *markdown) tea.Cmd {
	return func() tea.Msg {
		if md.markdownType != LocalDoc {
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

func filterMarkdowns(m stashModel) tea.Cmd {
	return func() tea.Msg {
		if m.filterInput.Value() == "" || !m.isFiltering() {
			return filteredMarkdownMsg(m.getFilterableMarkdowns()) // return everything
		}

		targets := []string{}
		mds := m.getFilterableMarkdowns()

		for _, t := range mds {
			targets = append(targets, t.filterValue)
		}

		ranks := fuzzy.Find(m.filterInput.Value(), targets)
		sort.Stable(ranks)

		filtered := []*markdown{}
		for _, r := range ranks {
			filtered = append(filtered, mds[r.Index])
		}

		return filteredMarkdownMsg(filtered)
	}
}

// ETC

// fetchMarkdown performs the actual I/O for loading markdown from the sever.
func fetchMarkdown(cc *charm.Client, id int, t DocType) (*markdown, error) {
	var md *charm.Markdown
	var err error

	switch t {
	case StashedDoc, ConvertedDoc:
		md, err = cc.GetStashMarkdown(id)
	case NewsDoc:
		md, err = cc.GetNewsMarkdown(id)
	default:
		err = fmt.Errorf("unknown markdown type: %s", t)
	}

	if err != nil {
		return nil, err
	}

	return &markdown{
		markdownType: t,
		Markdown:     *md,
	}, nil
}

// Delete a markdown from a slice of markdowns.
func deleteMarkdown(markdowns []*markdown, target *markdown) ([]*markdown, error) {
	index := -1

	for i, v := range markdowns {
		switch target.markdownType {
		case ConvertedDoc:
			if v.markdownType == ConvertedDoc && v.localPath == target.localPath {
				index = i
			}
		case StashedDoc:
			if v.ID == target.ID {
				index = i
			}
		default:
			return nil, fmt.Errorf("%s documents cannot be deleted", target.markdownType)
		}
	}

	if index == -1 {
		err := fmt.Errorf("could not find markdown to delete")
		if debug {
			log.Println(err)
		}
		return nil, err
	}

	return append(markdowns[:index], markdowns[index+1:]...), nil
}
