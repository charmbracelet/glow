package gold

import (
	"io"

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
		Pre:   pre,
		Token: string(node.Literal),
		Style: Paragraph,
	}
	return el.Render(w, node, tr)
}
