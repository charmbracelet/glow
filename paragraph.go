package gold

import (
	"bytes"
	"io"
	"strings"

	"github.com/muesli/reflow"
)

type ParagraphElement struct {
	InsideList bool
}

func (e *ParagraphElement) Render(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack
	var rules ElementStyle

	if e.InsideList {
		// list item
		rules = ctx.style[List]
	} else {
		rules = ctx.style[Paragraph]
		_, _ = w.Write([]byte("\n"))
		be := BlockElement{
			Block: &bytes.Buffer{},
			Style: cascadeStyle(bs.Current().Style, rules, true),
		}
		bs.Push(be)
	}

	renderText(w, bs.Current().Style, rules.Prefix)
	return nil
}

func (e *ParagraphElement) Finish(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack
	rules := bs.Current().Style

	keepNewlines := false
	if e.InsideList {
		// remove indent & margin for list items
		rules = bs.Current().Style
		keepNewlines = true
	}

	renderText(bs.Current().Block, rules, rules.Suffix)

	mw := NewMarginWriter(ctx, w, rules)
	if len(strings.TrimSpace(bs.Current().Block.String())) > 0 {
		flow := reflow.NewReflow(int(bs.Width(ctx)))
		flow.KeepNewlines = keepNewlines
		_, _ = flow.Write(bs.Current().Block.Bytes())
		flow.Close()

		_, err := mw.Write(flow.Bytes())
		if err != nil {
			return err
		}
		_, _ = mw.Write([]byte("\n"))
	}

	bs.Current().Block.Reset()
	if !e.InsideList {
		bs.Pop()
	}
	return nil
}
