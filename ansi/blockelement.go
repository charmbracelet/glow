package ansi

import (
	"bytes"
	"io"

	"github.com/muesli/reflow"
)

type BlockElement struct {
	Block   *bytes.Buffer
	Style   StyleBlock
	Margin  bool
	Newline bool
}

func (e *BlockElement) Render(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack
	bs.Push(*e)

	renderText(w, bs.Parent().Style.StylePrimitive, e.Style.BlockPrefix)
	renderText(bs.Current().Block, bs.Current().Style.StylePrimitive, e.Style.Prefix)
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

		if e.Newline {
			_, err = mw.Write([]byte("\n"))
			if err != nil {
				return err
			}
		}
	} else {
		_, err := bs.Parent().Block.Write(bs.Current().Block.Bytes())
		if err != nil {
			return err
		}
	}

	renderText(w, bs.Current().Style.StylePrimitive, e.Style.Suffix)
	renderText(w, bs.Parent().Style.StylePrimitive, e.Style.BlockSuffix)

	bs.Current().Block.Reset()
	bs.Pop()
	return nil
}
