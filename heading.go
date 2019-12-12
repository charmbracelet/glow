package gold

import (
	"bytes"
	"io"

	"github.com/muesli/reflow"
)

type HeadingElement struct {
	Level int
	First bool
}

func (e *HeadingElement) Render(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack
	rules := ctx.style[Heading]

	switch e.Level {
	case 1:
		rules = cascadeStyles(false, rules, ctx.style[H1])
	case 2:
		rules = cascadeStyles(false, rules, ctx.style[H2])
	case 3:
		rules = cascadeStyles(false, rules, ctx.style[H3])
	case 4:
		rules = cascadeStyles(false, rules, ctx.style[H4])
	case 5:
		rules = cascadeStyles(false, rules, ctx.style[H5])
	case 6:
		rules = cascadeStyles(false, rules, ctx.style[H6])
	}

	if !e.First {
		renderText(w, bs.Current().Style, "\n")
	}

	be := BlockElement{
		Block: &bytes.Buffer{},
		Style: cascadeStyle(bs.Current().Style, rules, true),
	}
	bs.Push(be)

	renderText(w, bs.Parent().Style, rules.Prefix)
	renderText(w, bs.Current().Style, rules.StyledPrefix)
	return nil
}

func (e *HeadingElement) Finish(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack
	rules := bs.Current().Style

	var indent uint
	var margin uint
	if rules.Indent != nil {
		indent = *rules.Indent
	}
	if rules.Margin != nil {
		margin = *rules.Margin
	}

	iw := &IndentWriter{
		Indent: indent + margin,
		IndentFunc: func(wr io.Writer) {
			renderText(w, bs.Parent().Style, " ")
		},
		Forward: &AnsiWriter{
			Forward: w,
		},
	}

	flow := reflow.NewReflow(int(bs.Width(ctx) - indent - margin*2))
	_, err := flow.Write(bs.Current().Block.Bytes())
	if err != nil {
		return err
	}
	flow.Close()

	_, err = iw.Write(flow.Bytes())
	if err != nil {
		return err
	}

	renderText(w, bs.Current().Style, rules.StyledSuffix)
	renderText(w, bs.Parent().Style, rules.Suffix)

	bs.Current().Block.Reset()
	bs.Pop()
	return nil
}
