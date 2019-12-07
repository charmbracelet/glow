package gold

import (
	"bytes"
	"io"
)

type DocumentElement struct {
	Width uint
}

func (e *DocumentElement) Render(w io.Writer, ctx RenderContext) error {
	rules := ctx.style[Document]

	be := BlockElement{
		Block: &bytes.Buffer{},
		Style: rules,
	}
	ctx.blockStack.Push(be)

	renderText(ctx.blockStack.Current().Block, rules, rules.Prefix)
	return nil
}

func (e *DocumentElement) Finish(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack
	rules := ctx.style[Document]

	var indent uint
	var margin uint
	if rules.Indent != nil {
		indent = *rules.Indent
	}
	if rules.Margin != nil {
		margin = *rules.Margin
	}
	suffix := rules.Suffix

	pw := &PaddingWriter{
		Padding: e.Width - margin,
		PadFunc: func(wr io.Writer) {
			renderText(w, rules, " ")
		},
		Forward: &AnsiWriter{
			Forward: w,
		},
	}
	iw := &IndentWriter{
		Indent: indent + margin,
		Forward: &AnsiWriter{
			Forward: pw,
		},
	}
	_, err := iw.Write(bs.Current().Block.Bytes())
	if err != nil {
		return err
	}
	renderText(iw, rules, suffix)

	bs.Current().Block.Reset()
	bs.Pop()

	return nil
}
