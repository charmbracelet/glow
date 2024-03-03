package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/editor"
)

type editorFinishedMsg struct{ err error }

func openEditor(path string, viewport_optional ...*viewport.Model) tea.Cmd {
	var lineNumber uint = 0
	if len(viewport_optional) == 1 {
		vp := viewport_optional[0]
		lineNumber = uint(vp.YOffset + vp.Height)
	}

	cb := func(err error) tea.Msg {
		return editorFinishedMsg{err}
	}

	editorCmd, err := editor.Cmd("Glow", path, editor.OpenAtLine(lineNumber))
	if err != nil {
		return func() tea.Msg {
			return errMsg{err}
		}
	}

	return tea.ExecProcess(editorCmd, cb)
}
