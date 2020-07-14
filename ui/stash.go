package ui

import (
	"fmt"
	"log"
	"math"
	"os"
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
	"github.com/muesli/gitcha"
	te "github.com/muesli/termenv"
)

const (
	stashViewItemHeight        = 3
	stashViewTopPadding        = 5
	stashViewBottomPadding     = 4
	stashViewHorizontalPadding = 6
	setNotePromptText          = "Memo: "
)

// MSG

type gotStashMsg []*charm.Markdown
type gotNewsMsg []*charm.Markdown
type fetchedMarkdownMsg *markdown
type deletedStashedItemMsg int
type fileWalkFinishedMsg []string

// MODEL

// markdownType allows us to differentiate between the types of markdown
// documents we're dealing with, namely stuff the user stashed versus news.
type markdownType int

const (
	userMarkdown markdownType = iota
	newsMarkdown
	localFile
)

// markdown wraps charm.Markdown so we can differentiate between stashed items
// and news.
type markdown struct {
	markdownType markdownType
	*charm.Markdown
}

// Sort documents by date in descending order
type markdownsByCreatedAtDesc []*markdown

// Sort implementation for MarkdownByCreatedAt
func (m markdownsByCreatedAtDesc) Len() int           { return len(m) }
func (m markdownsByCreatedAtDesc) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }
func (m markdownsByCreatedAtDesc) Less(i, j int) bool { return m[i].CreatedAt.After(*m[j].CreatedAt) }

type stashState int

const (
	stashStateInit stashState = iota
	stashStateReady
	stashStatePromptDelete
	stashStateLoadingDocument
	stashStateSettingNote
)

type stashLoadedState byte

func (s stashLoadedState) done() bool {
	return s&loadedStash != 0 && s&loadedNews != 0
}

const (
	loadedStash stashLoadedState = 1 << iota
	loadedNews
)

type stashModel struct {
	cc             *charm.Client
	state          stashState
	markdowns      []*markdown
	spinner        spinner.Model
	noteInput      textinput.Model
	terminalWidth  int
	terminalHeight int
	loaded         stashLoadedState // what's loaded? we find out with bitmasking
	loading        bool             // are we currently loading something?
	fullyLoaded    bool             // Have we loaded everything from the server?

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
	if len(m.markdowns) == 0 || len(m.markdowns) <= m.markdownIndex() {
		return nil
	}
	return m.markdowns[m.markdownIndex()]
}

// addDocuments adds markdown documents to the model
func (m *stashModel) addMarkdowns(mds ...*markdown) {
	m.markdowns = append(m.markdowns, mds...)
	sort.Sort(markdownsByCreatedAtDesc(m.markdowns))
	m.paginator.SetTotalPages(len(m.markdowns))
}

// INIT

func stashInit(cc *charm.Client) (stashModel, tea.Cmd) {
	s := spinner.NewModel()
	s.Type = spinner.Dot
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
		cc:        cc,
		spinner:   s,
		noteInput: ni,
		page:      1,
		paginator: p,
	}

	return m, tea.Batch(
		loadLocalFiles,
		loadStash(m),
		loadNews(m),
		spinner.Tick(s),
	)
}

// UPDATE

func stashUpdate(msg tea.Msg, m stashModel) (stashModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {

	// We've received a list of local markdowns
	case fileWalkFinishedMsg:
		if len(msg) > 0 {
			now := time.Now()
			for _, mdPath := range msg {
				m.markdowns = append(m.markdowns, &markdown{
					markdownType: localFile,
					Markdown: &charm.Markdown{
						Note:      mdPath,
						CreatedAt: &now,
					},
				})
			}
		}

	// Stash results have come in from the server
	case gotStashMsg:
		m.loading = false

		if len(msg) == 0 {
			// If the server comes back with nothing then we've got everything
			m.fullyLoaded = true
		} else {
			docs := wrapMarkdowns(userMarkdown, msg)
			m.addMarkdowns(docs...)
		}

		m.loaded |= loadedStash
		if m.loaded.done() {
			m.state = stashStateReady
		}

	case gotNewsMsg:
		if len(msg) > 0 {
			docs := wrapMarkdowns(newsMarkdown, msg)
			m.addMarkdowns(docs...)
		}

		m.loaded |= loadedNews
		if m.loaded.done() {
			m.state = stashStateReady
		}

	case spinner.TickMsg:
		if m.state == stashStateInit || m.state == stashStateLoadingDocument {
			m.spinner, cmd = spinner.Update(msg, m.spinner)
			cmds = append(cmds, cmd)
		}

	// A note was set on a document. This may have happened in the pager, so
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

				cmds = append(cmds,
					loadMarkdown(m.cc, md.ID, md.markdownType),
					spinner.Tick(m.spinner),
				)

			// Set note
			case "m":
				md := m.selectedMarkdown()
				isUserMarkdown := md.markdownType == userMarkdown
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
				isUserMarkdown := m.selectedMarkdown().markdownType == userMarkdown
				isValidState := m.state != stashStateSettingNote

				if isUserMarkdown && isValidState {
					m.state = stashStatePromptDelete
				}

			}
		}

		// Update paginator
		m.paginator, cmd = paginator.Update(msg, m.paginator)
		cmds = append(cmds, cmd)

		// Keep the index in bounds when paginating
		itemsOnPage := m.paginator.ItemsOnPage(len(m.markdowns))
		if m.index > itemsOnPage-1 {
			m.index = itemsOnPage - 1
		}

		// If we're on the last page and we haven't loaded everything, get
		// more stuff.
		if m.paginator.OnLastPage() && !m.loading && !m.fullyLoaded {
			m.page++
			m.loading = true
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
				cmd = saveDocumentNote(m.cc, md.ID, newNote)
				md.Note = newNote
				m.noteInput.Reset()
				m.state = stashStateReady
				return m, cmd
			}
		}

		// Update the text input component used to set notes
		m.noteInput, cmd = textinput.Update(msg, m.noteInput)
		cmds = append(cmds, cmd)

	}

	// If an item is being confirmed for delete, any key (other than the key
	// used for confirmation above) cancels the deletion

	return m, tea.Batch(cmds...)
}

// VIEW

func stashView(m stashModel) string {
	var s string
	switch m.state {
	case stashStateInit:
		s += " " + spinner.View(m.spinner) + " Loading stash..."
	case stashStateLoadingDocument:
		s += " " + spinner.View(m.spinner) + " Loading document..."
	case stashStateReady:
		fallthrough
	case stashStateSettingNote:
		fallthrough
	case stashStatePromptDelete:

		// We need to fill any empty height with newlines so the footer reaches
		// the bottom.
		numBlankLines := max(0, (m.terminalHeight-stashViewTopPadding-stashViewBottomPadding)%stashViewItemHeight)
		blankLines := ""
		if numBlankLines > 0 {
			blankLines = strings.Repeat("\n", numBlankLines)
		}

		var header string
		if len(m.markdowns) > 0 {
			header = "Here’s your markdown stash:"
		} else {
			header = "Nothing stashed yet. To stash you can " + common.Code("glow stash path/to/file.md") + "."
		}

		switch m.state {
		case stashStatePromptDelete:
			header = te.String("Delete this item? ").Foreground(common.Red.Color()).String() +
				te.String("(y/N)").Foreground(common.FaintRed.Color()).String()
		case stashStateSettingNote:
			header = te.String("Set the memo for this item?").Foreground(common.YellowGreen.Color()).String()
		}

		var pagination string
		if m.paginator.TotalPages > 1 {
			pagination = paginator.View(m.paginator)

			if !m.fullyLoaded {
				pagination += common.Subtle(" ···")
			}
		}

		s += fmt.Sprintf(
			"  %s\n\n  %s\n\n%s\n\n%s  %s\n\n  %s",
			glowLogoView(" Glow "),
			header,
			stashPopulatedView(m),
			blankLines,
			pagination,
			stashHelpView(m),
		)
	}
	return "\n" + indent(s, 1)
}

func glowLogoView(text string) string {
	return te.String(text).
		Bold().
		Foreground(glowLogoTextColor).
		Background(common.Fuschia.Color()).
		String()
}

func stashEmtpyView(m stashModel) string {
	return "Nothing stashed yet."
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
		h      []string
		md     = m.selectedMarkdown()
		isNews = md != nil && md.markdownType == newsMarkdown
	)

	if m.state == stashStateSettingNote {
		h = append(h, "enter: confirm", "esc: cancel")
	} else if m.state == stashStatePromptDelete {
		h = append(h, "y: delete", "n: cancel")
	} else {
		if len(m.markdowns) > 0 {
			h = append(h, "enter: open")
			h = append(h, "j/k, ↑/↓: choose")
		}
		if m.paginator.TotalPages > 1 {
			h = append(h, "h/l, ←/→: page")
		}
		if !isNews && len(m.markdowns) > 0 {
			h = append(h, []string{"x: delete", "m: set memo"}...)
		}
		h = append(h, []string{"esc: exit"}...)
	}
	return common.HelpView(h...)
}

// CMD

func loadLocalFiles() tea.Msg {
	cwd, err := os.Getwd()
	if err != nil {
		return errMsg(err)
	}

	// For now, wait to collect all the results before delivering them back
	var agg []string

	ch := gitcha.FindFileFromList(cwd, []string{"*.md"})
	for v := range ch {
		log.Println("found file", v)
		agg = append(agg, v)
	}

	return fileWalkFinishedMsg(agg)
}

func loadStash(m stashModel) tea.Cmd {
	return func() tea.Msg {
		stash, err := m.cc.GetStash(m.page)
		if err != nil {
			return errMsg(err)
		}
		return gotStashMsg(stash)
	}
}

func loadNews(m stashModel) tea.Cmd {
	return func() tea.Msg {
		news, err := m.cc.GetNews(1) // just fetch the first page
		if err != nil {
			return errMsg(err)
		}
		return gotNewsMsg(news)
	}
}

func loadMarkdown(cc *charm.Client, id int, t markdownType) tea.Cmd {
	return func() tea.Msg {
		var (
			md  *charm.Markdown
			err error
		)
		if t == userMarkdown {
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
