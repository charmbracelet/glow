package gold

import (
	"io"
	"strings"

	bf "gopkg.in/russross/blackfriday.v2"
)

type ItemElement struct {
}

func (e *ItemElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	l := 0
	n := node
	for n.Parent != nil && (n.Parent.Type == bf.List || n.Parent.Type == bf.Item) {
		if n.Parent.Type == bf.List {
			l++
		}
		n = n.Parent
	}

	el := &BaseElement{
		Pre:   strings.Repeat("  ", l-1) + "• ",
		Token: string(node.Literal),
		Style: Item,
	}
	return el.Render(w, node, tr)
}
