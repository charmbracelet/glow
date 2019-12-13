package gold

import (
	"bytes"
	"io"
)

type DocumentElement struct {
}

func (e *DocumentElement) Render(w io.Writer, ctx RenderContext) error {
	rules := ctx.styles.Document

	be := BlockElement{
		Block: &bytes.Buffer{},
		Style: rules,
	}
	ctx.blockStack.Push(be)

	renderText(ctx.blockStack.Current().Block, rules.StylePrimitive, rules.Prefix)
	return nil
}

func (e *DocumentElement) Finish(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack
	rules := ctx.styles.Document

	mw := NewMarginWriter(ctx, w, rules)
	_, err := mw.Write(bs.Current().Block.Bytes())
	if err != nil {
		return err
	}
	renderText(mw, rules.StylePrimitive, rules.Suffix)

	bs.Current().Block.Reset()
	bs.Pop()

	return nil
}
