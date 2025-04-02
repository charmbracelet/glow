package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/editor"
)

type editorFinishedMsg struct{ err error }

func openEditor(path string, lineno int) tea.Cmd {
	cb := func(err error) tea.Msg {
		return editorFinishedMsg{err}
	}
	cmd, err := editor.Cmd("Glow", path, editor.LineNumber(uint(lineno))) //nolint:gosec
	if err != nil {
		return func() tea.Msg { return cb(err) }
	}
	return tea.ExecProcess(cmd, cb)
}
