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

	var indent uint
	var margin uint
	keepNewlines := false
	if e.InsideList {
		// remove indent & margin for list items
		rules = bs.Current().Style
		keepNewlines = true
	}

	if rules.Indent != nil {
		indent = *rules.Indent
	}
	if rules.Margin != nil {
		margin = *rules.Margin
	}

	suffix := rules.Suffix
	renderText(bs.Current().Block, rules, suffix)

	pw := &PaddingWriter{
		Padding: bs.Width(ctx),
		PadFunc: func(wr io.Writer) {
			renderText(w, rules, " ")
		},
		Forward: &AnsiWriter{
			Forward: w,
		},
	}
	iw := &IndentWriter{
		Indent: indent + margin,
		IndentFunc: func(wr io.Writer) {
			renderText(w, bs.Parent().Style, " ")
		},
		Forward: &AnsiWriter{
			Forward: pw,
		},
	}

	if len(strings.TrimSpace(bs.Current().Block.String())) > 0 {
		flow := reflow.NewReflow(int(bs.Width(ctx)))
		flow.KeepNewlines = keepNewlines
		_, _ = flow.Write(bs.Current().Block.Bytes())
		flow.Close()

		_, err := iw.Write(flow.Bytes())
		if err != nil {
			return err
		}
		_, _ = pw.Write([]byte("\n"))
	}

	bs.Current().Block.Reset()
	if !e.InsideList {
		bs.Pop()
	}
	return nil
}
