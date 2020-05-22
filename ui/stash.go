package ui

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/boba"
	"github.com/charmbracelet/boba/paginator"
	"github.com/charmbracelet/boba/spinner"
	"github.com/charmbracelet/boba/textinput"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/dustin/go-humanize"
	"github.com/muesli/reflow/indent"
	te "github.com/muesli/termenv"
)

const (
	stashViewItemHeight        = 3
	stashViewTopPadding        = 5
	stashViewBottomPadding     = 4
	stashViewHorizontalPadding = 6
	setNotePromptText          = "Memo: "
)

var (
	faintGreen  = common.NewColorPair("#2B4A3F", "#ABE5D1")
	green       = common.NewColorPair("#04B575", "#04B575")
	dullYellow  = common.NewColorPair("#9BA92F", "#6CCCA9") // renders light green on light backgrounds
	dullFuchsia = common.NewColorPair("#AD58B4", "#F9ACFF")
)

// MSG

type stashErrMsg error
type stashSpinnerTickMsg struct{}
type gotStashMsg []*charm.Markdown
type gotNewsMsg []*charm.Markdown
type fetchedMarkdownMsg *markdown
type deletedStashedItemMsg int

// MODEL

// markdownType allows us to differentiate between the types of markdown
// documents we're dealing with, namely stuff the user stashed versus news.
type markdownType int

const (
	userMarkdown markdownType = iota
	newsMarkdown
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
	err            error
	state          stashState
	documents      []*markdown
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
	m.paginator.SetTotalPages(len(m.documents))

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

// INIT

func stashInit(cc *charm.Client) (stashModel, boba.Cmd) {
	s := spinner.NewModel()
	s.Type = spinner.Dot
	s.ForegroundColor = common.SpinnerColor
	s.CustomMsgFunc = func() boba.Msg { return stashSpinnerTickMsg{} }

	p := paginator.NewModel()
	p.Type = paginator.Dots
	p.InactiveDot = common.Subtle("•")

	ni := textinput.NewModel()
	ni.Prompt = te.String(setNotePromptText).Foreground(common.YellowGreen.Color()).String()
	ni.CursorColor = common.Fuschia.String()
	ni.CharLimit = noteCharacterLimit // totally arbitrary
	ni.Focus()

	m := stashModel{
		cc:        cc,
		spinner:   s,
		noteInput: ni,
		page:      1,
		paginator: p,
	}

	return m, boba.Batch(
		loadStash(m),
		loadNews(m),
		spinner.Tick(s),
	)
}

// UPDATE

func stashUpdate(msg boba.Msg, m stashModel) (stashModel, boba.Cmd) {
	var (
		cmd  boba.Cmd
		cmds []boba.Cmd
	)

	switch msg := msg.(type) {
	case stashErrMsg:
		m.err = msg

	// Stash results have come in from the server
	case gotStashMsg:
		m.loading = false

		// If the server comes back with nothing then we've got everything
		if len(msg) == 0 {
			m.fullyLoaded = true
			break
		}

		docs := wrapMarkdowns(userMarkdown, msg)
		sort.Sort(markdownsByCreatedAtDesc(docs)) // sort by date
		m.documents = append(m.documents, docs...)
		m.paginator.SetTotalPages(len(m.documents))

		m.loaded |= loadedStash
		if m.loaded.done() {
			m.state = stashStateReady
		}

	case gotNewsMsg:
		if len(msg) > 0 {
			docs := wrapMarkdowns(newsMarkdown, msg)
			sort.Sort(markdownsByCreatedAtDesc(docs))
			m.documents = append(m.documents, docs...)
			m.paginator.SetTotalPages(len(m.documents))
		}

		m.loaded |= loadedNews
		if m.loaded.done() {
			m.state = stashStateReady
		}

	case stashSpinnerTickMsg:
		if m.state == stashStateInit || m.state == stashStateLoadingDocument {
			m.spinner, cmd = spinner.Update(msg, m.spinner)
			cmds = append(cmds, cmd)
		}

	// A note was set on a document. This may have happened in the pager, so
	// we'll find the corresponding document here and update accordingly.
	case noteSavedMsg:
		for i := range m.documents {
			if m.documents[i].ID == msg.ID {
				m.documents[i].Note = msg.Note
			}
		}
	}

	switch m.state {
	case stashStateReady:
		if msg, ok := msg.(boba.KeyMsg); ok {
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
					m.index = m.paginator.ItemsOnPage(len(m.documents)) - 1
				}

			case "j":
				fallthrough
			case "down":
				itemsOnPage := m.paginator.ItemsOnPage(len(m.documents))
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
				// Load the document from the server. We'll handle the message
				// that comes back in the main update function.
				m.state = stashStateLoadingDocument
				doc := m.documents[m.markdownIndex()]

				cmds = append(cmds,
					loadMarkdown(m.cc, doc.ID, doc.markdownType),
					spinner.Tick(m.spinner),
				)

			// Set note
			case "n":
				if m.state != stashStateSettingNote && m.state != stashStatePromptDelete {
					m.state = stashStateSettingNote
					m.noteInput.SetValue(m.documents[m.markdownIndex()].Note)
					m.noteInput.CursorEnd()
					return m, textinput.Blink(m.noteInput)
				}

			// Prompt for deletion
			case "x":
				isUserMarkdown := m.documents[m.markdownIndex()].markdownType == userMarkdown
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
		itemsOnPage := m.paginator.ItemsOnPage(len(m.documents))
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
		if msg, ok := msg.(boba.KeyMsg); ok {
			switch msg.String() {

			// Confirm deletion
			case "y":
				if m.state != stashStatePromptDelete {
					break
				}

				i := m.markdownIndex()
				id := m.documents[i].ID

				// Delete optimistically and remove the stashed item
				// before we've received a success response.
				m.documents = append(m.documents[:i], m.documents[i+1:]...)

				// Update pagination
				m.paginator.SetTotalPages(len(m.documents))
				m.paginator.Page = min(m.paginator.Page, m.paginator.TotalPages-1)

				// Set state and delete
				m.state = stashStateReady
				return m, deleteStashedItem(m.cc, id)

			default:
				m.state = stashStateReady
			}
		}

	case stashStateSettingNote:

		if msg, ok := msg.(boba.KeyMsg); ok {
			switch msg.String() {
			case "q":
				fallthrough
			case "esc":
				// Cancel note
				m.state = stashStateReady
				m.noteInput.Reset()
			case "enter":
				// Set new note
				doc := m.documents[m.markdownIndex()]
				newNote := m.noteInput.Value()
				cmd = saveDocumentNote(m.cc, doc.ID, newNote)
				doc.Note = newNote
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

	return m, boba.Batch(cmds...)
}

// VIEW

func stashView(m stashModel) string {
	if m.err != nil {
		return "\n" + m.err.Error()
	}

	var s string
	switch m.state {
	case stashStateInit:
		s += spinner.View(m.spinner) + " Loading stash..."
	case stashStateLoadingDocument:
		s += spinner.View(m.spinner) + " Loading document..."
	case stashStateReady:
		fallthrough
	case stashStateSettingNote:
		fallthrough
	case stashStatePromptDelete:
		if len(m.documents) == 0 {
			s += stashEmtpyView(m)
			break
		}

		// We need to fill any empty height with newlines so the footer reaches
		// the bottom.
		numBlankLines := max(0, (m.terminalHeight-stashViewTopPadding-stashViewBottomPadding)%stashViewItemHeight)
		blankLines := ""
		if numBlankLines > 0 {
			blankLines = strings.Repeat("\n", numBlankLines)
		}

		header := "Here’s your markdown stash:"
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
			"%s\n\n%s\n\n%s\n\n%s%s\n\n%s",
			glowLogoView(" Glow "),
			header,
			stashPopulatedView(m),
			blankLines,
			pagination,
			stashHelpView(m),
		)
	}
	return "\n" + indent.String(s, 2)
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
	var s string

	start, end := m.paginator.GetSliceBounds(len(m.documents))
	docs := m.documents[start:end]

	for i, v := range docs {
		state := markdownStateNormal
		if i == m.index {
			switch m.state {
			case stashStatePromptDelete:
				state = markdownStateDeleting
			case stashStateSettingNote:
				state = markdownStateSettingNote
			default:
				state = markdownStateSelected
			}
		}
		s += stashListItemView(*v).render(m, state) + "\n\n"
	}
	s = strings.TrimSpace(s) // trim final newlines

	// If there aren't enough items to fill up this page (always the last page)
	// then we need to add some newlines to fill up the space to push the
	// footer stuff down elsewhere.
	itemsOnPage := m.paginator.ItemsOnPage(len(m.documents))
	if itemsOnPage < m.paginator.PerPage {
		n := (m.paginator.PerPage - itemsOnPage) * stashViewItemHeight
		s += strings.Repeat("\n", n)
	}

	return s
}

func stashHelpView(m stashModel) string {
	var (
		h      []string
		isNews bool = m.documents[m.markdownIndex()].markdownType == newsMarkdown
	)

	if m.state == stashStateSettingNote {
		h = append(h, "enter: confirm", "esc: cancel")
	} else if m.state == stashStatePromptDelete {
		h = append(h, "y: delete", "n: cancel")
	} else {
		h = append(h, "enter: open")
		if len(m.documents) > 0 {
			h = append(h, "j/k, ↑/↓: choose")
		}
		if m.paginator.TotalPages > 1 {
			h = append(h, "h/l, ←/→: page")
		}
		if !isNews {
			h = append(h, []string{"x: delete"}...)
		}
		h = append(h, []string{"esc: exit"}...)
	}
	return common.HelpView(h...)
}

// markdownState is used in a deterministic fashion to aid rendering stash item
// views.
type markdownState int

const (
	markdownStateNormal markdownState = iota
	markdownStateSelected
	markdownStateDeleting
	markdownStateSettingNote
)

// stashListItemView contains methods for rendering an item as it appears in
// the stash view
type stashListItemView markdown

func (m stashListItemView) render(mdl stashModel, state markdownState) string {

	// General key color
	keyColor := common.NoColor
	line := common.VerticalLine(common.StateNormal)
	switch state {
	case markdownStateSettingNote:
		keyColor = common.YellowGreen
		line = common.VerticalLine(common.StateActive)
	case markdownStateSelected:
		keyColor = common.Fuschia
		line = common.VerticalLine(common.StateSelected)
	case markdownStateDeleting:
		keyColor = common.Red
		line = common.VerticalLine(common.StateDeleting)
	default:
		if m.markdownType == newsMarkdown {
			keyColor = common.Green
		}
	}

	titleKey := "#" + strconv.Itoa(m.ID)
	if state == markdownStateSettingNote {
		titleKey = textinput.View(mdl.noteInput)
	} else {
		if m.markdownType == newsMarkdown {
			titleKey = "System Announcement"
		}
		if m.Note != "" {
			titleKey += ":"
		}
	}

	dateKey := "Stashed:"
	if m.markdownType == newsMarkdown {
		dateKey = "Posted:"
	}

	var titleVal string
	if state != markdownStateSettingNote {
		titleVal = truncate(m.Note, mdl.terminalWidth-(stashViewHorizontalPadding*2)-len(titleKey))
		titleVal = m.title(titleVal, state)
	}

	titleKey = te.String(titleKey).Foreground(keyColor.Color()).String()
	dateKey = te.String(dateKey).Foreground(keyColor.Color()).String()
	dateVal := m.date(state)

	var s string
	s += fmt.Sprintf("%s %s %s\n", line, titleKey, titleVal)
	s += fmt.Sprintf("%s %s %s", line, dateKey, dateVal)
	return s
}

func (m stashListItemView) date(state markdownState) string {
	c := common.Indigo
	if state == markdownStateDeleting {
		c = common.FaintRed
	} else if state == markdownStateSettingNote {
		c = dullYellow
	}
	s := relativeTime(*m.CreatedAt)
	return te.String(s).Foreground(c.Color()).String()
}

func (m stashListItemView) title(title string, state markdownState) string {
	if title == "" {
		return ""
	}
	c := common.Indigo
	if state == markdownStateDeleting {
		c = common.Red
	}
	return te.String(title).Foreground(c.Color()).String()
}

// CMD

func loadStash(m stashModel) boba.Cmd {
	return func() boba.Msg {
		stash, err := m.cc.GetStash(m.page)
		if err != nil {
			return stashErrMsg(err)
		}
		return gotStashMsg(stash)
	}
}

func loadNews(m stashModel) boba.Cmd {
	return func() boba.Msg {
		news, err := m.cc.GetNews(1) // just fetch the first page
		if err != nil {
			return stashErrMsg(err)
		}
		return gotNewsMsg(news)
	}
}

func loadMarkdown(cc *charm.Client, id int, t markdownType) boba.Cmd {
	return func() boba.Msg {
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
			return stashErrMsg(err)
		}
		return fetchedMarkdownMsg(&markdown{
			markdownType: userMarkdown,
			Markdown:     md,
		})
	}
}

func deleteStashedItem(cc *charm.Client, id int) boba.Cmd {
	return func() boba.Msg {
		err := cc.DeleteMarkdown(id)
		if err != nil {
			return stashErrMsg(err)
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
	s := str
	if len(str) > num {
		if num > 1 {
			num -= 1
		}
		s = str[0:num] + "…"
	}
	return s
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
