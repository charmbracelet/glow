package gold

import (
	"io"

	bf "gopkg.in/russross/blackfriday.v2"
)

type LinkElement struct {
}

func (e *LinkElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	if node.LastChild != nil {
		if node.LastChild.Type == bf.Image {
			el := tr.NewElement(node.LastChild)
			el.Renderer.Render(w, node.LastChild, tr)
		}
		if len(node.LastChild.Literal) > 0 {
			el := &BaseElement{
				Token: string(node.LastChild.Literal),
				Style: LinkText,
			}
			el.Render(w, node.LastChild, tr)
		}
	}
	if len(node.LinkData.Destination) > 0 {
		el := &BaseElement{
			Token:  resolveRelativeURL(tr.BaseURL, string(node.LinkData.Destination)),
			Prefix: " (",
			Suffix: ")",
			Style:  Link,
		}
		el.Render(w, node, tr)
	}

	return nil
}
