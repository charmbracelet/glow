package gold

import (
	"bytes"
	"io"

	"github.com/muesli/reflow"
)

type ListElement struct {
	Nested bool
}

func (e *ListElement) Render(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack
	rules := ctx.style[List]

	if !e.Nested {
		_, _ = w.Write([]byte("\n"))
	}

	be := BlockElement{
		Block: &bytes.Buffer{},
		Style: cascadeStyle(bs.Current().Style, rules, true),
	}
	bs.Push(be)

	return nil
}

func (e *ListElement) Finish(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack
	rules := bs.Current().Style

	renderText(bs.Current().Block, rules, rules.Suffix)

	_, err := NewMarginWriter(ctx, w, rules).Write(
		reflow.Bytes(bs.Current().Block.Bytes(), int(bs.Width(ctx))))
	if err != nil {
		return err
	}

	bs.Current().Block.Reset()
	bs.Pop()
	return nil
}
