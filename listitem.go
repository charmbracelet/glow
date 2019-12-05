package gold

import (
	"io"
	"strconv"

	bf "gopkg.in/russross/blackfriday.v2"
)

type ItemElement struct {
}

func (e *ItemElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	var el *BaseElement

	if node.ListData.ListFlags&bf.ListTypeOrdered > 0 {
		var l int64
		n := node
		for n.Prev != nil && (n.Prev.Type == bf.Item) {
			l++
			n = n.Prev
		}

		el = &BaseElement{
			Token:  string(node.Literal),
			Style:  tr.style[Enumeration],
			Prefix: strconv.FormatInt(l+1, 10),
		}
	} else {
		el = &BaseElement{
			Token: string(node.Literal),
			Style: tr.style[Item],
		}
	}

	return el.Render(w, node, tr)
}
