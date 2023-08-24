package editor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const defaultEditor = "nano"

// Cmd returns a *exec.Cmd editing the given path with $EDITOR or nano if no
// $EDITOR is set.
func Cmd(path string) (*exec.Cmd, error) {
	if os.Getenv("SNAP_REVISION") != "" {
		return nil, fmt.Errorf("glow is sandboxed, and thus cannot open an editor. To continue using this feature, please install glow using another option")
	}
	editor, args := getEditor()
	return exec.Command(editor, append(args, path)...), nil
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
