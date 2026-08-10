package ui

import (
	"strings"
	"unicode/utf8"
)

func highlightFocusedLink(rendered string, links []followableLink, focused int) string {
	if focused < 0 || focused >= len(links) {
		return rendered
	}

	printable, offsets := printableRunesAndOffsets(rendered)
	if len(printable) == 0 {
		return rendered
	}
	printableStr := string(printable)

	type span struct {
		start int
		end   int
		ok    bool
	}

	spans := make([]span, len(links))
	searchFrom := 0
	for i, l := range links {
		label := strings.TrimSpace(l.Label)
		if label == "" || searchFrom >= len(printableStr) {
			continue
		}

		relIdx := strings.Index(printableStr[searchFrom:], label)
		if relIdx < 0 {
			continue
		}
		byteIdx := searchFrom + relIdx
		searchFrom = byteIdx + len(label)

		startRune := utf8.RuneCountInString(printableStr[:byteIdx])
		endRune := startRune + utf8.RuneCountInString(label)
		if startRune < 0 || endRune > len(offsets)-1 {
			continue
		}

		startByte := offsets[startRune]
		endByte := offsets[endRune]
		if startByte < 0 || endByte < startByte || endByte > len(rendered) {
			continue
		}

		spans[i] = span{start: startByte, end: endByte, ok: true}
	}

	s := spans[focused]
	if !s.ok {
		return rendered
	}

	const (
		reverseOn  = "\x1b[7m"
		reverseOff = "\x1b[27m"
	)

	var b strings.Builder
	b.Grow(len(rendered) + len(reverseOn) + len(reverseOff))
	b.WriteString(rendered[:s.start])
	b.WriteString(reverseOn)
	b.WriteString(rendered[s.start:s.end])
	b.WriteString(reverseOff)
	b.WriteString(rendered[s.end:])
	return b.String()
}

func printableRunesAndOffsets(s string) ([]rune, []int) {
	var (
		runes   []rune
		offsets []int
	)

	for i := 0; i < len(s); {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) {
				c := s[i]
				i++
				if c >= 0x40 && c <= 0x7E {
					break
				}
			}
			continue
		}

		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			r = rune(s[i])
			size = 1
		}

		runes = append(runes, r)
		offsets = append(offsets, i)
		i += size
	}

	offsets = append(offsets, len(s))

	return runes, offsets
}
