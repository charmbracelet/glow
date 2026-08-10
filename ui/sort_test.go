package ui

import (
	"testing"
)

func TestSortMarkdowns(t *testing.T) {
	tests := []struct {
		name      string
		notes     []string
		wantNotes []string
	}{
		{
			name:      "alphabetical sort",
			notes:     []string{"cherry", "apple", "banana"},
			wantNotes: []string{"apple", "banana", "cherry"},
		},
		{
			name:      "empty slice",
			notes:     []string{},
			wantNotes: []string{},
		},
		{
			name:      "single item",
			notes:     []string{"only"},
			wantNotes: []string{"only"},
		},
		{
			name:      "already sorted",
			notes:     []string{"a", "b", "c"},
			wantNotes: []string{"a", "b", "c"},
		},
		{
			name:      "reverse order",
			notes:     []string{"c", "b", "a"},
			wantNotes: []string{"a", "b", "c"},
		},
		{
			name:      "duplicate notes stable",
			notes:     []string{"b", "a", "b", "a"},
			wantNotes: []string{"a", "a", "b", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mds := make([]*markdown, len(tt.notes))
			for i, n := range tt.notes {
				mds[i] = &markdown{Note: n}
			}

			sortMarkdowns(mds)

			for i, md := range mds {
				if md.Note != tt.wantNotes[i] {
					t.Errorf("index %d: got Note=%q, want %q", i, md.Note, tt.wantNotes[i])
				}
			}
		})
	}
}
