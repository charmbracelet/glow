package ui

import "testing"

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
