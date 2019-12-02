package gold

import (
	"io"
	"strings"
)

type IndentWriter struct {
	Indent  uint
	Forward io.Writer

	skipIndent bool
}

// Write is used to write content to the indent buffer.
func (w *IndentWriter) Write(b []byte) (int, error) {
	for _, c := range string(b) {
		if !w.skipIndent {
			_, err := w.Forward.Write([]byte(strings.Repeat(" ", int(w.Indent))))
			if err != nil {
				return 0, err
			}
			w.skipIndent = true
		}

		_, err := w.Forward.Write([]byte{byte(c)})
		if err != nil {
			return 0, err
		}
		if c == '\n' {
			// end of current line
			w.skipIndent = false
		}
	}

	return len(b), nil
}
