package gold

import (
	"bytes"
	"io"

	"github.com/muesli/reflow"
)

type BlockElement struct {
	Block  *bytes.Buffer
	Style  ElementStyle
	Margin bool
}

func (e *BlockElement) Render(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack
	bs.Push(*e)

	renderText(w, bs.Parent().Style, e.Style.Prefix)
	renderText(w, bs.Current().Style, e.Style.StyledPrefix)
	return nil
}

func (e *BlockElement) Finish(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack

	if e.Margin {
		mw := NewMarginWriter(ctx, w, bs.Current().Style)
		_, err := mw.Write(
			reflow.Bytes(bs.Current().Block.Bytes(), int(bs.Width(ctx))))
		if err != nil {
			return err
		}
		mw.Write([]byte("\n"))
	} else {
		_, err := bs.Parent().Block.Write(bs.Current().Block.Bytes())
		if err != nil {
			return err
		}
	}

	renderText(w, bs.Current().Style, e.Style.StyledSuffix)
	renderText(w, bs.Parent().Style, e.Style.Suffix)

	bs.Current().Block.Reset()
	bs.Pop()
	return nil
}
