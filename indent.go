package gold

import (
	"strings"
)

type IndentWriter struct {
	Forward *AnsiWriter
	Indent  uint

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
			w.Forward.RestoreAnsi()

			w.skipIndent = true
		}

		if c == '\n' {
			// end of current line
			w.skipIndent = false
			w.Forward.ResetAnsi()
		}
		_, err := w.Forward.Write([]byte(string(c)))
		if err != nil {
			return 0, err
		}
	}

	return len(b), nil
}
