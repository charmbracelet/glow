package gold

import (
	"io"
	"strings"
)

type AnsiWriter struct {
	Forward io.Writer

	ansi       bool
	ansiseq    string
	lastseq    string
	seqchanged bool
}

// Write is used to write content to the ANSI buffer.
func (w *AnsiWriter) Write(b []byte) (int, error) {
	for _, c := range string(b) {
		if c == '\x1B' {
			// ANSI escape sequence
			w.ansi = true
			w.seqchanged = true
			w.ansiseq += string(c)
		} else if w.ansi {
			w.ansiseq += string(c)
			if (c >= 0x41 && c <= 0x5a) || (c >= 0x61 && c <= 0x7a) {
				// ANSI sequence terminated
				w.ansi = false

				_, _ = w.Forward.Write([]byte(w.ansiseq))
				if strings.HasSuffix(w.ansiseq, "[0m") {
					// reset sequence
					w.lastseq = ""
				} else if strings.HasSuffix(w.ansiseq, "m") {
					// color code
					w.lastseq = w.ansiseq
				}
				w.ansiseq = ""
			}
		} else {
			_, err := w.Forward.Write([]byte(string(c)))
			if err != nil {
				return 0, err
			}
		}
	}

	return len(b), nil
}

func (w *AnsiWriter) LastSequence() string {
	return w.lastseq
}

func (w *AnsiWriter) ResetAnsi() {
	if !w.seqchanged {
		return
	}
	_, _ = w.Forward.Write([]byte("\x1b[0m"))
}

func (w *AnsiWriter) RestoreAnsi() {
	_, _ = w.Forward.Write([]byte(w.lastseq))
}
