package main

import (
	"errors"
	"testing"
)

func TestResolveWidth(t *testing.T) {
	tt := []struct {
		name            string
		isTerminal      bool
		configuredWidth uint
		flagChanged     bool
		detectedWidth   int
		detectErr       error
		want            uint
	}{
		{
			name:            "explicit width keeps configured value",
			isTerminal:      true,
			configuredWidth: 40,
			flagChanged:     true,
			detectedWidth:   180,
			want:            40,
		},
		{
			name:            "explicit zero width is preserved",
			isTerminal:      true,
			configuredWidth: 0,
			flagChanged:     true,
			detectedWidth:   180,
			want:            0,
		},
		{
			name:            "auto width uses detected terminal width",
			isTerminal:      true,
			configuredWidth: 0,
			flagChanged:     false,
			detectedWidth:   100,
			want:            100,
		},
		{
			name:            "auto width no longer caps terminal width",
			isTerminal:      true,
			configuredWidth: 0,
			flagChanged:     false,
			detectedWidth:   180,
			want:            180,
		},
		{
			name:            "auto width falls back when detection fails",
			isTerminal:      true,
			configuredWidth: 0,
			flagChanged:     false,
			detectErr:       errors.New("boom"),
			want:            80,
		},
		{
			name:            "non-tty fallback remains 80",
			isTerminal:      false,
			configuredWidth: 0,
			flagChanged:     false,
			want:            80,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveWidth(tc.isTerminal, tc.configuredWidth, tc.flagChanged, func() (int, error) {
				return tc.detectedWidth, tc.detectErr
			})

			if got != tc.want {
				t.Fatalf("resolveWidth() = %d, want %d", got, tc.want)
			}
		})
	}
}
