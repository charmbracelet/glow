package flow

import (
	"io"
)

// repeatReader generates a repeating pattern without pre-allocating memory
type repeatReader struct {
	pattern []byte
	limit   int64
	read    int64
}

func (r *repeatReader) Read(p []byte) (n int, err error) {
	if r.read >= r.limit {
		return 0, io.EOF
	}

	remaining := r.limit - r.read
	toRead := int64(len(p))
	if toRead > remaining {
		toRead = remaining
	}

	for i := int64(0); i < toRead; i++ {
		p[i] = r.pattern[(r.read+i)%int64(len(r.pattern))]
	}

	n = int(toRead)
	r.read += toRead
	return n, nil
}

// discardWriter counts bytes written but discards the data
type discardWriter struct {
	written int64
}

func (w *discardWriter) Write(p []byte) (int, error) {
	w.written += int64(len(p))
	return len(p), nil
}
