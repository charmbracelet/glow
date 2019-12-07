package gold

import (
	"io"

	"github.com/muesli/reflow"
)

type HeadingElement struct {
	Width uint
	Text  string
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

	flow := reflow.NewReflow(int(e.Width) -
		int(indent) - int(margin*2) -
		int(bs.Indent()) - int(bs.Margin())*2)

	var pre string
	if !e.First {
		pre = "\n"
	}
	el := &BaseElement{
		Prefix: pre,
		Token:  string(e.Text),
		Style:  rules,
	}
	err := el.Render(flow, ctx)
	if err != nil {
		return err
	}

	flow.Close()
	_, err = iw.Write(flow.Bytes())
	return err
}
