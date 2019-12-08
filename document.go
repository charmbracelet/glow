package gold

import (
	"bytes"
	"io"
)

type DocumentElement struct {
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

	mw := NewMarginWriter(ctx, w, rules)
	_, err := mw.Write(bs.Current().Block.Bytes())
	if err != nil {
		return err
	}
	renderText(mw, rules, rules.Suffix)

	bs.Current().Block.Reset()
	bs.Pop()

	return nil
}
