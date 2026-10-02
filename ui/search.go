package ui

import (
	"regexp"
	"strings"

	"charm.land/glow/v3/utils"
	"charm.land/lipgloss/v2"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
)

// osc66ST is the string terminator used by kitty's OSC 66 text-sizing
// protocol (see utils/textsize.go, which this package deliberately doesn't
// import from since these constants/helpers are unexported there).
const osc66ST = "\x1b\\"

// osc66Pattern matches a single kitty text-sizing (OSC 66) sequence as
// emitted by utils.ApplyTextSizing: ESC ] 66 ; <meta> ; <payload> ESC \.
// The payload is guaranteed not to contain further escape sequences --
// utils/textsize.go's sizedChunk strips control characters (including ESC)
// from heading text before wrapping it -- so matching up to the next ESC is
// safe and unambiguous. Go's regexp package interprets \x1b as the escape
// byte directly, so this is written as a plain raw string.
var osc66Pattern = regexp.MustCompile(`\x1b\]66;([^;\x1b]*);([^\x1b]*)\x1b\\`)

// searchMatch is one match location, in the coordinate system
// lipgloss.StyleRanges and viewport.Model.EnsureVisible expect: a zero-based
// line index, and a [colStart, colEnd) cell-width column range within that
// line's plain (ANSI-stripped) text.
type searchMatch struct {
	line             int
	colStart, colEnd int
}

// unwrapTextSizingPayloads replaces each OSC 66 text-sizing sequence with
// just its visible text payload. Generic ANSI strippers (including
// xansi.Strip) don't know this kitty protocol embeds real, visible heading
// text inside its payload -- they delete the whole sequence, payload
// included. Without this step, any heading rendered with text sizing
// enabled (GLOW_TEXT_SIZING=on, or auto-detected kitty support) becomes
// entirely invisible to search.
func unwrapTextSizingPayloads(content string) string {
	if !strings.Contains(content, "\x1b]66;") {
		return content
	}
	return osc66Pattern.ReplaceAllString(content, "$2")
}

// findMatches returns every case-insensitive, literal occurrence of query
// within the plain-text rendering of content (content with ANSI escape
// sequences stripped, and OSC 66 text-sizing payloads unwrapped to plain
// text), in document order.
//
// Matches are computed entirely within the stripped domain, one line at a
// time, deliberately avoiding viewport.Model.SetHighlights: that API has a
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
	lines := strings.Split(xansi.Strip(unwrapTextSizingPayloads(content)), "\n")
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
//
// byteStart/byteEnd come from regexp matches, which only guarantee alignment
// to rune boundaries, not grapheme cluster boundaries -- a multi-rune
// cluster (e.g. "e" + a combining accent, forming one visual "é") can have a
// match end partway through it. Snapping outward (colStart to the start of
// the cluster containing byteStart, colEnd to the end of the cluster
// containing byteEnd) rather than requiring exact equality keeps the
// resulting range valid -- colEnd >= colStart -- for every match, instead of
// silently leaving colEnd at its zero value when byteEnd never lines up
// exactly with a cluster boundary. An invalid (colEnd < colStart) range
// passed to lipgloss.StyleRanges doesn't just fail to highlight: it
// corrupts the rendered line, duplicating it in full (see lipgloss/v2's
// ranges.go -- StyleRanges sets lastIdx = rng.End unconditionally, so an End
// smaller than a prior Start rewinds lastIdx backward, and the final
// TruncateLeft(s, lastIdx, "") then re-emits content already written).
func byteRangeToCellRange(line string, byteStart, byteEnd int) (colStart, colEnd int) {
	bytePos, cellPos := 0, 0
	gr := uniseg.NewGraphemes(line)
	for gr.Next() {
		if bytePos <= byteStart {
			colStart = cellPos
		}
		bytePos += len(gr.Str())
		cellPos += max(1, gr.Width())
		if bytePos >= byteEnd {
			colEnd = cellPos
			break
		}
	}
	return colStart, colEnd
}

// lineToken is one piece of a rendered line, produced by tokenizeRenderedLine:
// either raw bytes to pass through unchanged (any escape sequence other than
// an OSC 66 text-sizing wrapper), or a run of visible text. A visible token
// whose osc66Meta is non-empty originated as the payload of an OSC 66
// sequence and must be re-wrapped with the same meta string to preserve its
// size when reassembled.
type lineToken struct {
	text      string
	visible   bool
	osc66Meta string
}

// tokenizeRenderedLine splits a rendered line into alternating escape and
// visible tokens, treating OSC 66 (kitty text-sizing) payloads as visible
// text rather than opaque escape sequences. This is what lets us bake search
// highlights into a line that contains a text-sized heading: lipgloss's own
// ansi-stripping (used internally by lipgloss.StyleRanges) has the same
// blind spot as xansi.Strip above and would misplace or drop highlights on
// such a line, so those lines are handled via highlightLineWithTextSizing
// instead of lipgloss.StyleRanges.
func tokenizeRenderedLine(line string) []lineToken {
	var tokens []lineToken
	for len(line) > 0 {
		i := strings.IndexByte(line, '\x1b')
		if i < 0 {
			tokens = append(tokens, lineToken{text: line, visible: true})
			break
		}
		if i > 0 {
			tokens = append(tokens, lineToken{text: line[:i], visible: true})
			line = line[i:]
		}
		if loc := osc66Pattern.FindStringSubmatchIndex(line); loc != nil && loc[0] == 0 {
			meta := line[loc[2]:loc[3]]
			payload := line[loc[4]:loc[5]]
			tokens = append(tokens, lineToken{text: payload, visible: true, osc66Meta: meta})
			line = line[loc[1]:]
			continue
		}
		seq := utils.ScanEscape(line)
		tokens = append(tokens, lineToken{text: seq})
		line = line[len(seq):]
	}
	return tokens
}

// highlightLineWithTextSizing bakes the given (non-overlapping, left-to-
// right ordered) highlight ranges into a rendered line that contains one or
// more OSC 66 text-sizing sequences, used in place of lipgloss.StyleRanges
// for such lines (see tokenizeRenderedLine for why). Matched visible text
// that came from an OSC 66 payload is re-wrapped in its own OSC 66 sequence
// (preserving the original meta/size) with the highlight style applied
// around it, so the terminal still renders it at the correct size.
func highlightLineWithTextSizing(line string, ranges []lipgloss.Range) string {
	tokens := tokenizeRenderedLine(line)

	var b strings.Builder
	rangeIdx := 0
	cellPos := 0

	for _, tok := range tokens {
		if !tok.visible {
			b.WriteString(tok.text)
			continue
		}

		// Accumulate consecutive graphemes that share the same highlight
		// state (plain, or highlighted by the same range) into one run, so
		// e.g. a multi-character match gets wrapped in a single OSC 66
		// sequence rather than one per grapheme.
		var run strings.Builder
		runHighlighted := false
		runRangeIdx := -1
		flush := func() {
			if run.Len() == 0 {
				return
			}
			wrapped := wrapVisible(run.String(), tok.osc66Meta)
			if runHighlighted {
				wrapped = ranges[runRangeIdx].Style.Render(wrapped)
			}
			b.WriteString(wrapped)
			run.Reset()
		}

		gr := uniseg.NewGraphemes(tok.text)
		for gr.Next() {
			cluster := gr.Str()
			width := max(1, gr.Width())

			for rangeIdx < len(ranges) && cellPos >= ranges[rangeIdx].End {
				rangeIdx++
			}
			inRange := rangeIdx < len(ranges) &&
				cellPos >= ranges[rangeIdx].Start && cellPos < ranges[rangeIdx].End

			if inRange != runHighlighted || (inRange && rangeIdx != runRangeIdx) {
				flush()
				runHighlighted = inRange
				runRangeIdx = rangeIdx
			}
			run.WriteString(cluster)
			cellPos += width
		}
		flush()
	}

	return b.String()
}

// wrapVisible re-wraps text in an OSC 66 text-sizing sequence using meta,
// or returns it unchanged if meta is empty (meaning it was already plain
// text, not an OSC 66 payload).
func wrapVisible(text, meta string) string {
	if meta == "" {
		return text
	}
	return "\x1b]66;" + meta + ";" + text + osc66ST
}
