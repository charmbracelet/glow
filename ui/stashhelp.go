package ui

import (
	"fmt"
	"strings"

	lib "github.com/charmbracelet/charm/ui/common"
	"github.com/muesli/reflow/ansi"
)

// helpEntry is a entry in a help menu containing values for a keystroke and
// it's associated action.
type helpEntry struct{ key, val string }

// helpColumn is a group of helpEntries which will be rendered into a column.
type helpColumn []helpEntry

// newHelpColumn creates a help column from pairs of string arguments
// represeting keys and values. If the arguments are not even (and therein
// not every key has a matching value) the function will panic.
func newHelpColumn(pairs ...string) (h helpColumn) {
	if len(pairs)%2 != 0 {
		panic("help text group must have an even number of items")
	}

	for i := 0; i < len(pairs); i = i + 2 {
		h = append(h, helpEntry{key: pairs[i], val: pairs[i+1]})
	}

	return
}

// render returns styled and formatted rows from keys and values.
func (h helpColumn) render(height int) (rows []string) {
	keyWidth, valWidth := h.maxWidths()

	for i := 0; i < height; i++ {
		var (
			b    = strings.Builder{}
			k, v string
		)
		if i < len(h) {
			k = h[i].key
			v = h[i].val

			switch k {
			case "s":
				k = greenFg(k)
				v = semiDimGreenFg(v)
			default:
				k = grayFg(k)
				v = midGrayFg(v)
			}
		}
		b.WriteString(k)
		b.WriteString(strings.Repeat(" ", keyWidth-ansi.PrintableRuneWidth(k))) // pad keys
		b.WriteString("  ")                                                     // gap
		b.WriteString(v)
		b.WriteString(strings.Repeat(" ", valWidth-ansi.PrintableRuneWidth(v))) // pad vals
		rows = append(rows, b.String())
	}

	return
}

// maxWidths returns the widest key and values in the column, respectively.
func (h helpColumn) maxWidths() (maxKey int, maxVal int) {
	for _, v := range h {
		kw := ansi.PrintableRuneWidth(v.key)
		vw := ansi.PrintableRuneWidth(v.val)
		if kw > maxKey {
			maxKey = kw
		}
		if vw > maxVal {
			maxVal = vw
		}
	}

	return
}

// helpView returns either the mini or full help view depending on the state of
// the model, as well as the total height of the help view.
func (m stashModel) helpView() (string, int) {
	numDocs := len(m.getVisibleMarkdowns())

	// Help for when we're filtering
	if m.filterState == filtering {
		var h []string

		switch numDocs {
		case 0:
			h = []string{"enter/esc", "cancel"}
		case 1:
			h = []string{"enter", "open", "esc", "cancel"}
		default:
			h = []string{"enter", "confirm", "esc", "cancel", "ctrl+j/ctrl+k ↑/↓", "choose"}
		}

		return m.renderHelp(h)
	}

	// Help for when we're interacting with a single document
	switch m.selectionState {
	case selectionSettingNote:
		return m.renderHelp([]string{"enter", "confirm", "esc", "cancel"}, []string{"q", "quit"})
	case selectionPromptingDelete:
		return m.renderHelp([]string{"y", "delete", "n", "cancel"}, []string{"q", "quit"})
	}

	var (
		isStashed     bool
		isStashable   bool
		navHelp       []string
		filterHelp    []string
		selectionHelp []string
		sectionHelp   []string
		appHelp       []string
	)

	if numDocs > 0 {
		md := m.selectedMarkdown()
		isStashed = md != nil && md.docType == StashedDoc
		isStashable = md != nil && md.docType == LocalDoc && m.online()
	}

	if numDocs > 0 && m.showFullHelp {
		navHelp = []string{"enter", "open", "j/k ↑/↓", "choose"}
	}

	if len(m.sections) > 1 {
		if m.showFullHelp {
			navHelp = append(navHelp, "tab/shift+tab", "section")
		} else {
			navHelp = append(navHelp, "tab", "section")
		}
	}

	if m.paginator().TotalPages > 1 {
		navHelp = append(navHelp, "h/l ←/→", "page")
	}

	// If we're browsing a filtered set
	if m.filterState == filterApplied {
		filterHelp = []string{"/", "edit search", "esc", "clear search"}
	} else {
		filterHelp = []string{"/", "find"}
	}

	if isStashed {
		selectionHelp = []string{"x", "delete", "m", "set memo"}
	} else if isStashable {
		selectionHelp = []string{"s", "stash"}
	}

	// If there are errors
	if m.err != nil {
		appHelp = append(appHelp, "!", "errors")
	}

	appHelp = append(appHelp, "q", "quit")

	// Detailed help
	if m.showFullHelp {
		if m.filterState != filtering {
			appHelp = append(appHelp, "?", "close help")
		}
		return m.renderHelp(navHelp, filterHelp, selectionHelp, sectionHelp, appHelp)
	}

	// Mini help
	if m.filterState != filtering {
		appHelp = append(appHelp, "?", "more")
	}
	return m.renderHelp(navHelp, filterHelp, selectionHelp, sectionHelp, appHelp)
}

// renderHelp returns the rendered help view and associated line height for
// the given groups of help items.
func (m stashModel) renderHelp(groups ...[]string) (string, int) {
	if m.showFullHelp {
		str := m.fullHelpView(groups...)
		numLines := strings.Count(str, "\n") + 1
		return str, numLines
	}
	return m.miniHelpView(concatStringSlices(groups...)...), 1
}

// Builds the help view from various sections pieces, truncating it if the view
// would otherwise wrap to two lines. Help view entires should come in as pairs,
// with the first being the key and the second being the help text.
func (m stashModel) miniHelpView(entries ...string) string {
	if len(entries) == 0 {
		return ""
	}

	var (
		truncationChar  = lib.Subtle("…")
		truncationWidth = ansi.PrintableRuneWidth(truncationChar)
	)

	var (
		next       string
		leftGutter = "  "
		maxWidth   = m.common.width -
			stashViewHorizontalPadding -
			truncationWidth -
			ansi.PrintableRuneWidth(leftGutter)
		s = leftGutter
	)

	for i := 0; i < len(entries); i = i + 2 {
		k := entries[i]
		v := entries[i+1]

		switch k {
		case "s":
			k = greenFg(k)
			v = semiDimGreenFg(v)
		default:
			k = grayFg(k)
			v = midGrayFg(v)
		}

		next = fmt.Sprintf("%s %s", k, v)

		if i < len(entries)-2 {
			next += dividerDot
		}

		// Only this (and the following) help text items if we have the
		// horizontal space
		if ansi.PrintableRuneWidth(s)+ansi.PrintableRuneWidth(next) >= maxWidth {
			s += truncationChar
			break
		}

		s += next
	}
	return s
}

func (m stashModel) fullHelpView(groups ...[]string) string {
	var (
		columns      []helpColumn
		tallestCol   int
		renderedCols [][]string // final rows grouped by column
	)

	// Get key/value pairs
	for _, g := range groups {
		if len(g) == 0 {
			continue // ignore empty columns
		}

		columns = append(columns, newHelpColumn(g...))
	}

	// Find the tallest column
	for _, c := range columns {
		if len(c) > tallestCol {
			tallestCol = len(c)
		}
	}

	// Build columns
	for _, c := range columns {
		renderedCols = append(renderedCols, c.render(tallestCol))
	}

	// Merge columns
	return mergeColumns(renderedCols...)
}

// Merge columns together to build the help view.
func mergeColumns(cols ...[]string) string {
	const minimumHeight = 3

	// Find the tallest column
	var tallestCol int
	for _, v := range cols {
		n := len(v)
		if n > tallestCol {
			tallestCol = n
		}
	}

	// Make sure the tallest column meets the minimum height
	if tallestCol < minimumHeight {
		tallestCol = minimumHeight
	}

	b := strings.Builder{}
	for i := 0; i < tallestCol; i++ {
		for j, col := range cols {
			if i >= len(col) {
				continue // skip if we're past the length of this column
			}
			if j == 0 {
				b.WriteString("  ") // gutter
			} else if j > 0 {
				b.WriteString("    ") // gap
			}
			b.WriteString(col[i])
		}
		if i < tallestCol-1 {
			b.WriteRune('\n')
		}
	}

	return b.String()
}

func concatStringSlices(s ...[]string) (agg []string) {
	for _, v := range s {
		agg = append(agg, v...)
	}
	return
}
