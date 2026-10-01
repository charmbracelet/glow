package ui

import (
	"strings"
	"testing"
)

func TestFindMatches(t *testing.T) {
	tests := []struct {
		name    string
		content string
		query   string
		want    []searchMatch
	}{
		{
			name:    "empty query returns no matches",
			content: "hello world",
			query:   "",
			want:    nil,
		},
		{
			name:    "no match",
			content: "hello world",
			query:   "xyz",
			want:    nil,
		},
		{
			name:    "single match",
			content: "hello world",
			query:   "world",
			want:    []searchMatch{{line: 0, colStart: 6, colEnd: 11}},
		},
		{
			name:    "multiple matches on one line",
			content: "hello world, hello again",
			query:   "hello",
			want: []searchMatch{
				{line: 0, colStart: 0, colEnd: 5},
				{line: 0, colStart: 13, colEnd: 18},
			},
		},
		{
			name:    "matches on different lines",
			content: "hello world\nhello again",
			query:   "hello",
			want: []searchMatch{
				{line: 0, colStart: 0, colEnd: 5},
				{line: 1, colStart: 0, colEnd: 5},
			},
		},
		{
			name:    "case-insensitive",
			content: "Hello WORLD hello",
			query:   "hello",
			want: []searchMatch{
				{line: 0, colStart: 0, colEnd: 5},
				{line: 0, colStart: 12, colEnd: 17},
			},
		},
		{
			name:    "wide runes are counted by cell width, not byte or rune count",
			content: "日本語 hello",
			query:   "hello",
			// "日本語" is 3 double-width runes (9 bytes, 3 runes, 6 cells)
			// followed by a space, so "hello" starts at cell 7.
			want: []searchMatch{{line: 0, colStart: 7, colEnd: 12}},
		},
		{
			name:    "match on a line with real ANSI SGR codes before it is attributed correctly",
			content: "\x1b[38;5;252msome unrelated text\x1b[m then hello there",
			query:   "hello",
			want:    []searchMatch{{line: 0, colStart: 25, colEnd: 30}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findMatches(tt.content, tt.query)
			if len(got) != len(tt.want) {
				t.Fatalf("findMatches(%q, %q) = %+v, want %+v", tt.content, tt.query, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("findMatches(%q, %q)[%d] = %+v, want %+v", tt.content, tt.query, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestFindMatchesDoesNotPanicOnAdversarialInput exercises inputs a hostile or
// careless user could plausibly type or paste into the search box, to
// confirm the query is always treated as a literal string (never
// interpreted as regex syntax) and that nothing here can panic regardless
// of query/content shape.
func TestFindMatchesDoesNotPanicOnAdversarialInput(t *testing.T) {
	adversarialQueries := []string{
		`.*`,                        // would match everything if treated as regex
		`(a+)+b`,                    // classic ReDoS pattern if treated as regex
		`[`,                         // unterminated character class if treated as regex
		`\`,                         // trailing backslash if treated as regex
		strings.Repeat("a", 10_000), // very long query
		"👍🏽日本語",                     // multi-byte / combining / wide runes
		"\x00\x01\x02",              // raw control bytes (shouldn't reach here in
		// practice, since textinput's sanitizer strips these before they
		// ever reach searchQuery, but findMatches itself must not assume
		// that and must not panic if ever called directly with them)
	}
	adversarialContent := []string{
		"",
		"\n\n\n",
		strings.Repeat("x", 50_000),
		"\x1b[38;5;252m\x1b[m\x1b[1m\x1b[m", // ANSI codes with no visible text
	}

	for _, content := range adversarialContent {
		for _, query := range adversarialQueries {
			findMatches(content, query) // must not panic
		}
	}

	// The regex-metacharacter queries must be treated as literal text, not
	// as regex syntax: searching for ".*" in content that does NOT
	// literally contain ".*" must yield zero matches, even though ".*" as
	// real regex would match the entire string.
	got := findMatches("hello world", ".*")
	if len(got) != 0 {
		t.Fatalf(`expected ".*" to be treated as a literal (non-matching) string, got matches: %+v`, got)
	}

	got = findMatches("literally .* here", ".*")
	if len(got) != 1 {
		t.Fatalf(`expected exactly one literal match of ".*", got: %+v`, got)
	}
}

// TestFindMatchesInsideTextSizedHeading is the regression test for a bug
// reported on the PR: with kitty text sizing enabled (GLOW_TEXT_SIZING=on),
// heading text is wrapped in OSC 66 sequences (\x1b]66;<meta>;<text>\x1b\),
// and generic ANSI strippers -- including xansi.Strip, which findMatches
// otherwise relies on -- don't know this protocol embeds visible text in
// its payload, so they delete it along with the escape codes. Without
// unwrapTextSizingPayloads, searching for text that only appears in a
// heading (e.g. "installation" when the only occurrence is "## Installation")
// finds zero matches.
func TestFindMatchesInsideTextSizedHeading(t *testing.T) {
	// A level-1/2 style heading: the whole heading text in one OSC 66
	// sequence (see utils/textsize.go's osc66Chunks).
	content := "\x1b[34;1m\x1b]66;s=2;Installation\x1b\\\x1b[m\nSome body text.\n"

	got := findMatches(content, "installation")
	if len(got) != 1 {
		t.Fatalf("expected 1 match inside the heading, got %d: %+v", len(got), got)
	}
	if got[0].line != 0 {
		t.Fatalf("expected the match on line 0 (the heading), got line %d", got[0].line)
	}
}

// TestFindMatchesInsideTightlyChunkedHeading covers level-3 headings, which
// utils/textsize.go's osc66TightChunks breaks into many small OSC 66
// sequences (one per up-to-4-cell-wide run, each with its own "w=" width
// parameter) rather than one sequence for the whole heading. A search term
// spanning multiple such chunks must still be found as a single contiguous
// match once unwrapped.
func TestFindMatchesInsideTightlyChunkedHeading(t *testing.T) {
	chunk := func(meta, text string) string {
		return "\x1b]66;" + meta + ";" + text + "\x1b\\"
	}
	// "Installation" split into several small OSC 66 chunks, as
	// osc66TightChunks would produce for an H3 heading.
	content := chunk("s=2:n=3:d=4:w=3", "Ins") +
		chunk("s=2:n=3:d=4:w=3", "tal") +
		chunk("s=2:n=3:d=4:w=3", "lat") +
		chunk("s=2:n=3:d=4:w=3", "ion") +
		"\n"

	got := findMatches(content, "installation")
	if len(got) != 1 {
		t.Fatalf("expected 1 match spanning the tight chunks, got %d: %+v", len(got), got)
	}
}

func TestUnwrapTextSizingPayloadsLeavesPlainContentUnchanged(t *testing.T) {
	plain := "hello \x1b[31mworld\x1b[0m\n"
	if got := unwrapTextSizingPayloads(plain); got != plain {
		t.Fatalf("expected content without OSC 66 sequences to be returned unchanged, got %q", got)
	}
}
