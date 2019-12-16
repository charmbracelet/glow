package ansi

import (
	"bytes"
	"io"
)

type StyleWriter struct {
	w     io.Writer
	buf   bytes.Buffer
	rules StylePrimitive
}

func NewStyleWriter(ctx RenderContext, w io.Writer, rules StylePrimitive) *StyleWriter {
	return &StyleWriter{
		w:     w,
		rules: rules,
	}
}

func (w *StyleWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

func (w *StyleWriter) Close() error {
	renderText(w.w, w.rules, w.buf.String())
	return nil
}
