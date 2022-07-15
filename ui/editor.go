package ui

import (
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const defaultEditor = "nano"

type editorFinishedMsg struct{ err error }

func openEditor(path string) tea.Cmd {
	editor, args := getEditor()
	cmd := exec.Command(editor, append(args, path)...)
	cb := func(err error) tea.Msg {
		return editorFinishedMsg{err}
	}
	return tea.ExecProcess(cmd, cb)
}

func getEditor() (string, []string) {
	editor := strings.Fields(os.Getenv("EDITOR"))
	if len(editor) > 1 {
		return editor[0], editor[1:]
	}
	if len(editor) == 1 {
		return editor[0], []string{}
	}
	return defaultEditor, []string{}
}
