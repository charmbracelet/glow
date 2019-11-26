package gold

import (
	"io"

	bf "gopkg.in/russross/blackfriday.v2"
)

type ImageElement struct {
}

func (e *ImageElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	if len(node.LastChild.Literal) > 0 {
		el := &BaseElement{
			Token: string(node.LastChild.Literal),
			Style: Image,
		}
		el.Render(w, node.LastChild, tr)
	}
	if len(node.LinkData.Destination) > 0 {
		el := &BaseElement{
			Token:  resolveRelativeURL(tr.BaseURL, string(node.LinkData.Destination)),
			Prefix: " [Image: ",
			Suffix: "]",
			Style:  Link,
		}
		el.Render(w, node, tr)
	}

	return nil
}
