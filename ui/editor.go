package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glow/editor"
)

type editorFinishedMsg struct{ err error }

func openEditor(path string) tea.Cmd {
	cb := func(err error) tea.Msg {
		return editorFinishedMsg{err}
	}
	return tea.ExecProcess(editor.Cmd(path), cb)
}
