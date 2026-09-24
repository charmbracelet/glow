package main

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/bidi"
)

var ansiEscRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

var bracketMirror = map[rune]rune{
	'(': ')', ')': '(',
	'[': ']', ']': '[',
	'{': '}', '}': '{',
	'<': '>', '>': '<',
}

func containsRTL(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Arabic, r) || unicode.Is(unicode.Hebrew, r) ||
			unicode.Is(unicode.Syriac, r) || unicode.Is(unicode.Thaana, r) {
			return true
		}
	}
	return false
}

type styledRune struct {
	r     rune
	style string
}

func bidiReorder(s string) string {
	if !containsRTL(s) {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = bidiReorderAnsiLine(line)
	}
	return strings.Join(lines, "\n")
}

func bidiReorderAnsiLine(line string) string {
	if len(line) == 0 {
		return line
	}

	// Preserve leading whitespace (margins), strip trailing (padding)
	leadingLen := 0
	for _, r := range line {
		if r == ' ' || r == '\t' {
			leadingLen++
		} else {
			break
		}
	}
	leading := line[:leadingLen]
	trimmed := strings.TrimRight(line[leadingLen:], " \t")
	if len(trimmed) == 0 {
		return line
	}

	chars, trailingAnsi := parseStyledRunes(trimmed)
	if len(chars) == 0 {
		return line
	}

	plainRunes := make([]rune, len(chars))
	for i, c := range chars {
		plainRunes[i] = c.r
	}
	plainStr := string(plainRunes)

	if !containsRTL(plainStr) {
		return line
	}

	var p bidi.Paragraph
	p.SetString(plainStr)
	ordering, err := p.Order()
	if err != nil {
		return line
	}

	// run.Pos() returns rune indices with inclusive end
	reordered := make([]styledRune, 0, len(chars))
	for i := 0; i < ordering.NumRuns(); i++ {
		run := ordering.Run(i)
		startIdx, endIdx := run.Pos()
		endIdx++

		if startIdx < 0 {
			startIdx = 0
		}
		if endIdx > len(chars) {
			endIdx = len(chars)
		}
		if startIdx >= endIdx {
			continue
		}

		runChars := make([]styledRune, endIdx-startIdx)
		copy(runChars, chars[startIdx:endIdx])

		if run.Direction() == bidi.RightToLeft {
			for l, r := 0, len(runChars)-1; l < r; l, r = l+1, r-1 {
				runChars[l], runChars[r] = runChars[r], runChars[l]
			}
			for k := range runChars {
				if mirrored, ok := bracketMirror[runChars[k].r]; ok {
					runChars[k].r = mirrored
				}
			}
		}

		reordered = append(reordered, runChars...)
	}

	if len(reordered) == 0 {
		return line
	}

	var b strings.Builder
	b.Grow(len(line))
	b.WriteString(leading)
	prevStyle := ""
	for _, c := range reordered {
		if c.style != prevStyle {
			b.WriteString(c.style)
			prevStyle = c.style
		}
		b.WriteRune(c.r)
	}
	if trailingAnsi != "" {
		b.WriteString(trailingAnsi)
	}

	return b.String()
}

func parseStyledRunes(s string) ([]styledRune, string) {
	var chars []styledRune
	currentStyle := ""
	trailingAnsi := ""

	matches := ansiEscRegex.FindAllStringIndex(s, -1)

	pos := 0
	for _, m := range matches {
		if pos < m[0] {
			for _, r := range s[pos:m[0]] {
				chars = append(chars, styledRune{r: r, style: currentStyle})
			}
		}
		currentStyle = s[m[0]:m[1]]
		pos = m[1]
	}

	if pos < len(s) {
		for _, r := range s[pos:] {
			chars = append(chars, styledRune{r: r, style: currentStyle})
		}
	}

	if pos == len(s) && len(matches) > 0 {
		trailingAnsi = currentStyle
	}

	return chars, trailingAnsi
}
