package ansi

import (
	"io"
)

type MarginWriter struct {
	w  io.Writer
	pw *PaddingWriter
	iw *IndentWriter
}

func NewMarginWriter(ctx RenderContext, w io.Writer, rules StyleBlock) *MarginWriter {
	bs := ctx.blockStack

	var indent uint
	var margin uint
	if rules.Indent != nil {
		indent = *rules.Indent
	}
	if rules.Margin != nil {
		margin = *rules.Margin
	}

	pw := &PaddingWriter{
		Padding: bs.Width(ctx),
		PadFunc: func(wr io.Writer) {
			renderText(w, rules.StylePrimitive, " ")
		},
		Forward: &AnsiWriter{
			Forward: w,
		},
	}
	iw := &IndentWriter{
		Indent: indent + margin,
		IndentFunc: func(wr io.Writer) {
			renderText(w, bs.Parent().Style.StylePrimitive, " ")
		},
		Forward: &AnsiWriter{
			Forward: pw,
		},
	}

	return &MarginWriter{
		w:  w,
		pw: pw,
		iw: iw,
	}
}

func (w *MarginWriter) Write(b []byte) (int, error) {
	return w.iw.Write(b)
}
