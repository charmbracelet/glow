package gold

import (
	"io"

	"github.com/muesli/reflow"
	bf "gopkg.in/russross/blackfriday.v2"
)

type ParagraphElement struct {
}

func (e *ParagraphElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	pre := "\n"
	if node.Prev == nil || (node.Parent != nil && node.Parent.Type == bf.Item) {
		pre = ""
	}

	el := &BaseElement{
		Prefix: pre,
		Token:  string(node.Literal),
		Style:  Paragraph,
	}
	return el.Render(w, node, tr)
}

func (e *ParagraphElement) Finish(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	var indent uint
	rules := tr.style[Paragraph]
	if rules != nil {
		indent = rules.Indent
	}
	iw := &IndentWriter{
		Indent:  indent,
		Forward: w,
	}

	_, err := iw.Write(reflow.ReflowBytes(tr.paragraph.Bytes(), tr.WordWrap-int(indent)))
	tr.paragraph.Reset()
	return err
}
