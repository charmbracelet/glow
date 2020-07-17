package ui

import (
	"errors"
	"fmt"
	"io/ioutil"
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
	runewidth "github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/wordwrap"
	te "github.com/muesli/termenv"
)

const (
	stashIndent                = 1
	stashViewItemHeight        = 3
	stashViewTopPadding        = 5
	stashViewBottomPadding     = 4
	stashViewHorizontalPadding = 6
	setNotePromptText          = "Memo: "
)

// MSG

type fetchedMarkdownMsg *markdown
type deletedStashedItemMsg int

// MODEL

// markdownType allows us to differentiate between the types of markdown
// documents we're dealing with, namely stuff the user stashed versus news.
type markdownType int

const (
	stashedMarkdown markdownType = iota
	newsMarkdown
	localMarkdown
)

// markdown wraps charm.Markdown so we can differentiate between stashed items
// and news.
type markdown struct {
	markdownType markdownType
	localPath    string // only relevent to local files
	*charm.Markdown
}

// Sort documents with local files first, then by date
type markdownsByLocalFirst []*markdown

func (m markdownsByLocalFirst) Len() int      { return len(m) }
func (m markdownsByLocalFirst) Swap(i, j int) { m[i], m[j] = m[j], m[i] }
func (m markdownsByLocalFirst) Less(i, j int) bool {
	iType := m[i].markdownType
	jType := m[j].markdownType

	// Local files come first
	if iType == localMarkdown && jType != localMarkdown {
		return true
	}
	if iType != localMarkdown && jType == localMarkdown {
		return false
	}

	// Both or neither are local files so sort by date descending
	return m[i].CreatedAt.After(*m[j].CreatedAt)
}

type loadedState byte

const (
	loadedStash loadedState = 1 << iota
	loadedNews
	loadedLocalFiles
)

func (s loadedState) done() bool {
	return s&loadedStash != 0 &&
		s&loadedNews != 0 &&
		s&loadedLocalFiles != 0
}

type stashState int

const (
	stashStateReady stashState = iota
	stashStatePromptDelete
	stashStateLoadingDocument
	stashStateSettingNote
	stashStateShowingError
)

type stashModel struct {
	cc                 *charm.Client
	state              stashState
	err                error
	markdowns          []*markdown
	spinner            spinner.Model
	noteInput          textinput.Model
	terminalWidth      int
	terminalHeight     int
	stashFullyLoaded   bool        // have we loaded everything from the server?
	loadingFromNetwork bool        // are we currently loading something from the network?
	loaded             loadedState // what's loaded? we find out with bitmasking

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
}

func (m *stashModel) setSize(width, height int) {
	m.terminalWidth = width
	m.terminalHeight = height

	// Update the paginator
	perPage := max(1, (m.terminalHeight-stashViewTopPadding-stashViewBottomPadding)/stashViewItemHeight)
	m.paginator.PerPage = perPage
	m.paginator.SetTotalPages(len(m.markdowns))

	m.noteInput.Width = m.terminalWidth - stashViewHorizontalPadding*2 - len(setNotePromptText)

	// Make sure the page stays in bounds
	if m.paginator.Page >= m.paginator.TotalPages-1 {
		m.paginator.Page = max(0, m.paginator.TotalPages-1)
	}
}

// markdownIndex returns the index of the currently selected markdown item.
func (m stashModel) markdownIndex() int {
	return m.paginator.Page*m.paginator.PerPage + m.index
}

// return the current selected markdown in the stash
func (m stashModel) selectedMarkdown() *markdown {
	i := m.markdownIndex()
	if i < 0 || len(m.markdowns) == 0 || len(m.markdowns) <= i {
		return nil
	}
	return m.markdowns[i]
}

// addDocuments adds markdown documents to the model
func (m *stashModel) addMarkdowns(mds ...*markdown) {
	if len(mds) > 0 {
		m.markdowns = append(m.markdowns, mds...)
		sort.Sort(markdownsByLocalFirst(m.markdowns))
		m.paginator.SetTotalPages(len(m.markdowns))
	}
}

func (m stashModel) countMarkdowns(t markdownType) (found int) {
	if len(m.markdowns) == 0 {
		return
	}
	for i := 0; i < len(m.markdowns); i++ {
		if m.markdowns[i].markdownType == t {
			found++
		}
	}
	return
}

// INIT

func newStashModel() stashModel {
	s := spinner.NewModel()
	s.Frames = spinner.Line
	s.ForegroundColor = common.SpinnerColor

	p := paginator.NewModel()
	p.Type = paginator.Dots
	p.InactiveDot = common.Subtle("•")

	ni := textinput.NewModel()
	ni.Prompt = te.String(setNotePromptText).Foreground(common.YellowGreen.Color()).String()
	ni.CursorColor = common.Fuschia.String()
	ni.CharLimit = noteCharacterLimit
	ni.Focus()

	m := stashModel{
		spinner:            s,
		noteInput:          ni,
		page:               1,
		paginator:          p,
		loadingFromNetwork: true,
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
		m.loaded |= loadedStash // still done, albeit unsuccessfully
		m.stashFullyLoaded = true
		m.loadingFromNetwork = false

	case newsLoadErrMsg:
		m.err = msg.err
		m.loaded |= loadedNews // still done, albeit unsuccessfully

	// We're finished searching for local files
	case localFileSearchFinished:
		m.loaded |= loadedLocalFiles

	// Stash results have come in from the server
	case gotStashMsg:
		// This doesn't mean the whole stash listing is loaded, but some we've
		// finished checking for the stash, at least, so mark the stash as
		// loaded here.
		m.loaded |= loadedStash

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
		m.loaded |= loadedNews
		if len(msg) > 0 {
			docs := wrapMarkdowns(newsMarkdown, msg)
			m.addMarkdowns(docs...)
		}

	case spinner.TickMsg:
		if !m.loaded.done() || m.loadingFromNetwork || m.state == stashStateLoadingDocument {
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
	}

	switch m.state {
	case stashStateReady:

		// Handle keys
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {

			case "k":
				fallthrough
			case "up":
				m.index--
				if m.index < 0 && m.paginator.Page == 0 {
					// Stop
					m.index = 0
				} else if m.index < 0 {
					// Go to previous page
					m.paginator.PrevPage()
					m.index = m.paginator.ItemsOnPage(len(m.markdowns)) - 1
				}

			case "j":
				fallthrough
			case "down":
				itemsOnPage := m.paginator.ItemsOnPage(len(m.markdowns))
				m.index++
				if m.index >= itemsOnPage && m.paginator.OnLastPage() {
					// Stop
					m.index = itemsOnPage - 1
				} else if m.index >= itemsOnPage {
					// Go to next page
					m.index = 0
					m.paginator.NextPage()
				}

			// Open document
			case "enter":
				if len(m.markdowns) == 0 {
					break
				}

				// Load the document from the server. We'll handle the message
				// that comes back in the main update function.
				m.state = stashStateLoadingDocument
				md := m.selectedMarkdown()

				if md.markdownType == localMarkdown {
					cmds = append(cmds, loadLocalMarkdown(md))
				} else {
					cmds = append(cmds, loadRemoteMarkdown(m.cc, md.ID, md.markdownType))
				}

				cmds = append(cmds, spinner.Tick(m.spinner))

			// Set note
			case "m":
				md := m.selectedMarkdown()
				isUserMarkdown := md.markdownType == stashedMarkdown
				isSettingNote := m.state == stashStateSettingNote
				isPromptingDelete := m.state == stashStatePromptDelete

				if isUserMarkdown && !isSettingNote && !isPromptingDelete {
					m.state = stashStateSettingNote
					m.noteInput.SetValue(md.Note)
					m.noteInput.CursorEnd()
					return m, textinput.Blink(m.noteInput)
				}

			// Prompt for deletion
			case "x":
				isUserMarkdown := m.selectedMarkdown().markdownType == stashedMarkdown
				isValidState := m.state != stashStateSettingNote

				if isUserMarkdown && isValidState {
					m.state = stashStatePromptDelete
				}

			case "!":
				if m.err != nil && m.state == stashStateReady {
					m.state = stashStateShowingError
					return m, nil
				}

			}
		}

		// Update paginator
		newPaginatorModel, cmd := paginator.Update(msg, m.paginator)
		m.paginator = newPaginatorModel
		cmds = append(cmds, cmd)

		// Keep the index in bounds when paginating
		itemsOnPage := m.paginator.ItemsOnPage(len(m.markdowns))
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

				i := m.markdownIndex()
				id := m.markdowns[i].ID

				// Delete optimistically and remove the stashed item
				// before we've received a success response.
				m.markdowns = append(m.markdowns[:i], m.markdowns[i+1:]...)

				// Update pagination
				m.paginator.SetTotalPages(len(m.markdowns))
				m.paginator.Page = min(m.paginator.Page, m.paginator.TotalPages-1)

				// Set state and delete
				m.state = stashStateReady
				return m, deleteStashedItem(m.cc, id)

			default:
				m.state = stashStateReady
			}
		}

	case stashStateSettingNote:
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "esc":
				// Cancel note
				m.state = stashStateReady
				m.noteInput.Reset()
			case "enter":
				// Set new note
				md := m.selectedMarkdown()
				newNote := m.noteInput.Value()
				cmd := saveDocumentNote(m.cc, md.ID, newNote)
				md.Note = newNote
				m.noteInput.Reset()
				m.state = stashStateReady
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
	case stashStateReady, stashStateSettingNote, stashStatePromptDelete:
		var (
			header string
		)

		loadingIndicator := ""
		if !m.loaded.done() || m.loadingFromNetwork {
			loadingIndicator = spinner.View(m.spinner)
		}

		// We need to fill any empty height with newlines so the footer reaches
		// the bottom.
		numBlankLines := max(0, (m.terminalHeight-stashViewTopPadding-stashViewBottomPadding)%stashViewItemHeight)
		blankLines := ""
		if numBlankLines > 0 {
			blankLines = strings.Repeat("\n", numBlankLines)
		}

		switch m.state {
		case stashStatePromptDelete:
			header = redFg("Delete this item? ") + faintRedFg("(y/N)")
		case stashStateSettingNote:
			header = yellowFg("Set the memo for this item?")
		}

		// Only draw the normal header if we're not using the header area for
		// something else (like a prompt)
		if header == "" {
			header = stashHeaderView(m)
		}

		var pagination string
		if m.paginator.TotalPages > 1 {
			pagination = paginator.View(m.paginator)

			if !m.stashFullyLoaded {
				pagination += common.Subtle(" ···")
			}
		}

		s += fmt.Sprintf(
			"  %s %s\n\n  %s\n\n%s\n\n%s  %s\n\n  %s",
			glowLogoView(" Glow "),
			loadingIndicator,
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
	loading := !m.loaded.done()
	noMarkdowns := len(m.markdowns) == 0

	// Still loading. We haven't found files, stashed items, or news yet.
	if loading && noMarkdowns {
		return common.Subtle("Looking for stuff...")
	}

	// Loading's finished. We didn't find anything, the stash is empty and
	// there's no news.
	if !loading && noMarkdowns {
		s := "Nothing found. Try running " + common.Code("glow") +
			" in a directory with markdown files, or stashing a file with " +
			common.Code("glow stash") + ". For more, see " + common.Code("glow help") + "."
		return wordwrap.String(s, stashIndent)
	}

	localItems := m.countMarkdowns(localMarkdown)
	stashedItems := m.countMarkdowns(stashedMarkdown)

	// Loading's finished and all we have is news.
	if localItems == 0 && stashedItems == 0 {
		return common.Subtle("No local or stashed markdown files found.")
	}

	// There are local and/or stashed files, so display counts.
	var s string
	if localItems > 0 {
		var plural string
		if localItems > 1 {
			plural = "s"
		}
		s += fmt.Sprintf("%d File%s", localItems, plural)
	}
	if stashedItems > 0 {
		var divider string
		if localItems > 0 {
			divider = " • "
		}
		s += fmt.Sprintf("%s%d Stashed", divider, stashedItems)
	}
	return common.Subtle(s)
}

func stashPopulatedView(m stashModel) string {
	var b strings.Builder

	if len(m.markdowns) > 0 {
		start, end := m.paginator.GetSliceBounds(len(m.markdowns))
		docs := m.markdowns[start:end]

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
	itemsOnPage := m.paginator.ItemsOnPage(len(m.markdowns))
	if itemsOnPage < m.paginator.PerPage {
		n := (m.paginator.PerPage - itemsOnPage) * stashViewItemHeight
		if len(m.markdowns) == 0 {
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
	)

	if len(m.markdowns) > 0 {
		md := m.selectedMarkdown()
		isStashed = md != nil && md.markdownType == stashedMarkdown
	}

	if m.state == stashStateSettingNote {
		h = append(h, "enter: confirm", "esc: cancel")
	} else if m.state == stashStatePromptDelete {
		h = append(h, "y: delete", "n: cancel")
	} else {
		if len(m.markdowns) > 0 {
			h = append(h, "enter: open")
		}
		if len(m.markdowns) > 1 {
			h = append(h, "j/k, ↑/↓: choose")
		}
		if m.paginator.TotalPages > 1 {
			h = append(h, "h/l, ←/→: page")
		}
		if isStashed && len(m.markdowns) > 0 {
			h = append(h, []string{"x: delete", "m: set memo"}...)
		}
		if m.err != nil {
			h = append(h, []string{"!: errors"}...)
		}
		h = append(h, []string{"esc: exit"}...)
	}
	return common.HelpView(h...)
}

// CMD

func loadRemoteMarkdown(cc *charm.Client, id int, t markdownType) tea.Cmd {
	return func() tea.Msg {
		var (
			md  *charm.Markdown
			err error
		)

		if t == stashedMarkdown {
			md, err = cc.GetStashMarkdown(id)
		} else {
			md, err = cc.GetNewsMarkdown(id)
		}

		if err != nil {
			return errMsg(err)
		}

		return fetchedMarkdownMsg(&markdown{
			markdownType: t,
			Markdown:     md,
		})
	}
}

func loadLocalMarkdown(md *markdown) tea.Cmd {
	return func() tea.Msg {
		if md.markdownType != localMarkdown {
			return errMsg(errors.New("could not load local file: not a local file"))
		}
		if md.localPath == "" {
			return errMsg(errors.New("could not load file: missing path"))
		}

		data, err := ioutil.ReadFile(md.localPath)
		if err != nil {
			return errMsg(err)
		}
		md.Body = string(data)
		return fetchedMarkdownMsg(md)
	}
}

func deleteStashedItem(cc *charm.Client, id int) tea.Cmd {
	return func() tea.Msg {
		err := cc.DeleteMarkdown(id)
		if err != nil {
			return errMsg(err)
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
			Markdown:     v,
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
	if now.Sub(then) < humanize.Week {
		return humanize.CustomRelTime(then, now, "ago", "from now", magnitudes)
	}
	return then.Format("02 Jan 2006 15:04 MST")
}
