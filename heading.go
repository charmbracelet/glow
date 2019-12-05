package gold

import (
	"fmt"
	"io"
	"strings"

	"github.com/muesli/reflow"
	bf "gopkg.in/russross/blackfriday.v2"
)

type HeadingElement struct {
}

func (e *HeadingElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	var indent uint
	var margin uint
	rules := tr.style[Heading]

	switch node.HeadingData.Level {
	case 1:
		rules = cascadeStyles(false, rules, tr.style[H1])
	case 2:
		rules = cascadeStyles(false, rules, tr.style[H2])
	case 3:
		rules = cascadeStyles(false, rules, tr.style[H3])
	case 4:
		rules = cascadeStyles(false, rules, tr.style[H4])
	case 5:
		rules = cascadeStyles(false, rules, tr.style[H5])
	case 6:
		rules = cascadeStyles(false, rules, tr.style[H6])
	}

	if rules != nil {
		indent = rules.Indent
		margin = rules.Margin
	}

	iw := &IndentWriter{
		Indent: indent + margin,
		IndentFunc: func(wr io.Writer) {
			renderText(w, tr.blockStack.Parent().Style, " ")
		},
		Forward: &AnsiWriter{
			Forward: w,
		},
	}

	flow := reflow.NewReflow(tr.WordWrap - int(indent) - int(margin*2) - int(tr.blockStack.Indent()) - int(tr.blockStack.Margin())*2)

	var pre string
	if node.Prev != nil {
		pre = "\n"
	}
	el := &BaseElement{
		Prefix: pre,
		Token:  fmt.Sprintf("%s %s", strings.Repeat("#", node.HeadingData.Level), node.FirstChild.Literal),
		Style:  Heading,
	}
	err := el.Render(flow, node, tr)
	if err != nil {
		return err
	}

	flow.Close()
	_, err = iw.Write(flow.Bytes())
	return err
}
