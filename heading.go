package gold

import (
	"io"

	"github.com/muesli/reflow"
	bf "gopkg.in/russross/blackfriday.v2"
)

type HeadingElement struct {
}

func (e *HeadingElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	ctx := tr.context
	bs := ctx.blockStack
	rules := ctx.style[Heading]

	switch node.HeadingData.Level {
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

	flow := reflow.NewReflow(tr.WordWrap -
		int(indent) - int(margin*2) -
		int(bs.Indent()) - int(bs.Margin())*2)

	var pre string
	if node.Prev != nil {
		pre = "\n"
	}
	el := &BaseElement{
		Prefix: pre,
		Token:  string(node.FirstChild.Literal),
		Style:  rules,
	}
	err := el.Render(flow, node, tr)
	if err != nil {
		return err
	}

	flow.Close()
	_, err = iw.Write(flow.Bytes())
	return err
}
