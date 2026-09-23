package ui

import (
	"regexp"
	"strings"

	xansi "github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
)

// searchMatch is one match location, in the coordinate system
// lipgloss.StyleRanges and viewport.Model.EnsureVisible expect: a zero-based
// line index, and a [colStart, colEnd) cell-width column range within that
// line's plain (ANSI-stripped) text.
type searchMatch struct {
	line             int
	colStart, colEnd int
}

// findMatches returns every case-insensitive, literal occurrence of query
// within the plain-text rendering of content (content with ANSI escape
// sequences stripped), in document order.
//
// Matches are computed entirely within the ANSI-stripped domain, one line at
// a time, deliberately avoiding viewport.Model.SetHighlights: that API has a
// correctness bug for content containing ANSI escape codes (its internal
// line-boundary detection reads bytes from the wrong string once escape
// codes are present, silently misattributing matches to the wrong line).
// Stripping ANSI never removes or adds '\n' characters, so line counts and
// order are identical between raw and stripped content, letting us map a
// match found in stripped line N back to a highlight baked into raw line N
// with no ambiguity.
func findMatches(content, query string) []searchMatch {
	if query == "" {
		return nil
	}

	re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(query))

	var matches []searchMatch
	lines := strings.Split(xansi.Strip(content), "\n")
	for lineIdx, line := range lines {
		for _, loc := range re.FindAllStringIndex(line, -1) {
			colStart, colEnd := byteRangeToCellRange(line, loc[0], loc[1])
			matches = append(matches, searchMatch{line: lineIdx, colStart: colStart, colEnd: colEnd})
		}
	}
	return matches
}

// byteRangeToCellRange converts a [byteStart, byteEnd) byte range within a
// plain-text line into a cell-width [colStart, colEnd) range, accounting for
// wide runes (e.g. CJK) the same way viewport's own highlighting logic does.
func byteRangeToCellRange(line string, byteStart, byteEnd int) (colStart, colEnd int) {
	bytePos, cellPos := 0, 0
	gr := uniseg.NewGraphemes(line)
	for gr.Next() {
		if bytePos == byteStart {
			colStart = cellPos
		}
		bytePos += len(gr.Str())
		cellPos += max(1, gr.Width())
		if bytePos == byteEnd {
			colEnd = cellPos
		}
	}
	return colStart, colEnd
}
