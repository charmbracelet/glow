package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/editor"
)

type editorFinishedMsg struct{ err error }

func openEditor(path string) tea.Cmd {
	cb := func(err error) tea.Msg {
		return editorFinishedMsg{err}
	}

	editor, err := editor.Cmd("Glow", path)
	if err != nil {
		return func() tea.Msg {
			return errMsg{err}
		}
	}
	return tea.ExecProcess(editor, cb)
}
