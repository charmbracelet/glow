package ui

import (
	"testing"
	"time"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"diacritics cafe", "café", "cafe", false},
		{"diacritics naive", "naïve", "naive", false},
		{"diacritics Munchen", "München", "Munchen", false},
		{"ASCII unchanged", "hello world", "hello world", false},
		{"empty string", "", "", false},
		{"mixed diacritics", "résumé.md", "resume.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("normalize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("normalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		when time.Time
		want string
	}{
		{
			name: "just now",
			when: now.Add(-10 * time.Second),
			want: "just now",
		},
		{
			name: "minutes ago",
			when: now.Add(-5 * time.Minute),
			want: "5 minutes ago",
		},
		{
			name: "hours ago",
			when: now.Add(-3 * time.Hour),
			want: "3 hours ago",
		},
		{
			name: "days ago",
			when: now.Add(-2 * 24 * time.Hour),
			want: "2 days ago",
		},
		{
			name: "old date uses formatted date",
			when: time.Date(2020, 1, 15, 10, 30, 0, 0, time.UTC),
			want: "15 Jan 2020 10:30 UTC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relativeTime(tt.when)
			if got != tt.want {
				t.Errorf("relativeTime() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildFilterValue(t *testing.T) {
	tests := []struct {
		name string
		note string
		want string
	}{
		{
			name: "plain text",
			note: "readme",
			want: "readme",
		},
		{
			name: "diacritics stripped",
			note: "café résumé",
			want: "cafe resume",
		},
		{
			name: "empty note",
			note: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := &markdown{Note: tt.note}
			md.buildFilterValue()
			if md.filterValue != tt.want {
				t.Errorf("buildFilterValue() filterValue = %q, want %q", md.filterValue, tt.want)
			}
		})
	}
}
