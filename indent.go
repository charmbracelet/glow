package gold

import (
	"io"
	"strings"
)

type IndentFunc = func(w io.Writer)

type IndentWriter struct {
	Forward    *AnsiWriter
	Indent     uint
	IndentFunc IndentFunc

	skipIndent bool
	ansi       bool
}

// Write is used to write content to the indent buffer.
func (w *IndentWriter) Write(b []byte) (int, error) {
	for _, c := range string(b) {
		if c == '\x1B' {
			// ANSI escape sequence
			w.ansi = true
		} else if w.ansi {
			if (c >= 0x41 && c <= 0x5a) || (c >= 0x61 && c <= 0x7a) {
				// ANSI sequence terminated
				w.ansi = false
			}
		} else {
			if !w.skipIndent {
				w.Forward.ResetAnsi()
				if w.IndentFunc != nil {
					for i := 0; i < int(w.Indent); i++ {
						w.IndentFunc(w.Forward)
					}
				} else {
					_, err := w.Forward.Write([]byte(strings.Repeat(" ", int(w.Indent))))
					if err != nil {
						return 0, err
					}
				}

				w.skipIndent = true
				w.Forward.RestoreAnsi()
			}

			if c == '\n' {
				// end of current line
				w.skipIndent = false
			}
		}

		_, err := w.Forward.Write([]byte(string(c)))
		if err != nil {
			return 0, err
		}
	}

	return len(b), nil
}
