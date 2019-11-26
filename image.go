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
		err := el.Render(w, node.LastChild, tr)
		if err != nil {
			return err
		}
	}
	if len(node.LinkData.Destination) > 0 {
		el := &BaseElement{
			Token:  resolveRelativeURL(tr.BaseURL, string(node.LinkData.Destination)),
			Prefix: " [Image: ",
			Suffix: "]",
			Style:  Link,
		}
		err := el.Render(w, node, tr)
		if err != nil {
			return err
		}
	}

	return nil
}
