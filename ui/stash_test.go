package ui

import (
	"testing"

	"charm.land/bubbles/v2/paginator"
	tea "charm.land/bubbletea/v2"
)

func TestStashPaginatorKeys(t *testing.T) {
	tests := []struct {
		name  string
		start int
		key   tea.KeyPressMsg
		want  int
	}{
		{"l", 0, tea.KeyPressMsg{Code: 'l', Text: "l"}, 1},
		{"right", 0, tea.KeyPressMsg{Code: tea.KeyRight}, 1},
		{"pgdown", 0, tea.KeyPressMsg{Code: tea.KeyPgDown}, 1},
		{"h", 1, tea.KeyPressMsg{Code: 'h', Text: "h"}, 0},
		{"left", 1, tea.KeyPressMsg{Code: tea.KeyLeft}, 0},
		{"pgup", 1, tea.KeyPressMsg{Code: tea.KeyPgUp}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newStashModel(&commonModel{styles: newStyles(true)})
			m.paginator().TotalPages = 3
			m.paginator().Page = tt.start

			m.handleDocumentBrowsing(tt.key)

			if got := m.paginator().Page; got != tt.want {
				t.Errorf("Page = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestStashPaginatorArabicFallback(t *testing.T) {
	m := newStashModel(&commonModel{styles: newStyles(true)})
	p := *m.paginator()
	p.TotalPages = 3
	p.Type = paginator.Arabic

	if got := p.View(); got != "1/3" {
		t.Errorf("View() = %q, want %q", got, "1/3")
	}
}
