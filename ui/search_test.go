package ui

import "testing"

func TestFindMatches(t *testing.T) {
	tests := []struct {
		name    string
		content string
		query   string
		want    [][]int
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
			want:    [][]int{{6, 11}},
		},
		{
			name:    "multiple matches",
			content: "hello world, hello again",
			query:   "hello",
			want:    [][]int{{0, 5}, {13, 18}},
		},
		{
			name:    "case-insensitive",
			content: "Hello WORLD hello",
			query:   "hello",
			want:    [][]int{{0, 5}, {12, 17}},
		},
		{
			name:    "match spanning a newline is not required to work, but must not panic",
			content: "foo\nbar",
			query:   "o\nb",
			want:    [][]int{{2, 5}},
		},
		{
			name:    "match adjacent to an ANSI escape sequence still matches the plain text run",
			content: "\x1b[35mhello\x1b[0m world",
			query:   "hello",
			want:    [][]int{{5, 10}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findMatches(tt.content, tt.query)
			if len(got) != len(tt.want) {
				t.Fatalf("findMatches(%q, %q) = %v, want %v", tt.content, tt.query, got, tt.want)
			}
			for i := range got {
				if got[i][0] != tt.want[i][0] || got[i][1] != tt.want[i][1] {
					t.Fatalf("findMatches(%q, %q) = %v, want %v", tt.content, tt.query, got, tt.want)
				}
			}
		})
	}
}
