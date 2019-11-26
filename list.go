package gold

import (
	"io"

	bf "gopkg.in/russross/blackfriday.v2"
)

type ListElement struct {
}

func (e *ListElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	pre := ""
	if node.Parent.Type != bf.Item {
		pre = "\n"
	}

	el := &BaseElement{
		Pre:   pre,
		Token: string(node.Literal),
		Style: List,
	}
	return el.Render(w, node, tr)
}
