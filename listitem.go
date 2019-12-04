package gold

import (
	"io"

	bf "gopkg.in/russross/blackfriday.v2"
)

type ItemElement struct {
}

func (e *ItemElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	el := &BaseElement{
		Token: string(node.Literal),
		Style: Item,
	}
	return el.Render(w, node, tr)
}
