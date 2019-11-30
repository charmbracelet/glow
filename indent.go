package gold

import (
	"io"
	"strings"
	"sync"
)

type IndentWriter struct {
	Indent  uint
	Forward io.Writer

	initialWrite sync.Once
}

// Write is used to write more content to the reflow buffer.
func (w *IndentWriter) Write(b []byte) (int, error) {
	w.initialWrite.Do(func() {
		w.Forward.Write([]byte(strings.Repeat(" ", int(w.Indent))))
	})

	for _, c := range string(b) {
		w.Forward.Write([]byte{byte(c)})
		if c == '\n' {
			// end of current line
			w.Forward.Write([]byte(strings.Repeat(" ", int(w.Indent))))
		} else {
			// any other character
		}
	}

	return len(b), nil
}

// Close will finish the reflow operation. Always call it before trying to
// retrieve the final result.
func (w *IndentWriter) Close() error {
	return nil
}
