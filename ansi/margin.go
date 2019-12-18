package ansi

import (
	"io"

	"github.com/muesli/reflow/ansi"
	"github.com/muesli/reflow/indent"
	"github.com/muesli/reflow/padding"
)

type MarginWriter struct {
	w  io.Writer
	pw *padding.Writer
	iw *indent.Writer
}

func NewMarginWriter(ctx RenderContext, w io.Writer, rules StyleBlock) *MarginWriter {
	bs := ctx.blockStack

	var indentation uint
	var margin uint
	if rules.Indent != nil {
		indentation = *rules.Indent
	}
	if rules.Margin != nil {
		margin = *rules.Margin
	}

	pw := &padding.Writer{
		Padding: bs.Width(ctx),
		PadFunc: func(wr io.Writer) {
			renderText(w, rules.StylePrimitive, " ")
		},
		Forward: &ansi.Writer{
			Forward: w,
		},
	}

	ic := " "
	if rules.IndentToken != nil {
		ic = *rules.IndentToken
	}
	iw := &indent.Writer{
		Indent: indentation + margin,
		IndentFunc: func(wr io.Writer) {
			renderText(w, bs.Parent().Style.StylePrimitive, ic)
		},
		Forward: &ansi.Writer{
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
