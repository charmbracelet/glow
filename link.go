package gold

import (
	"io"

	bf "gopkg.in/russross/blackfriday.v2"
)

type LinkElement struct {
}

func (e *LinkElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	ctx := tr.context

	var textRendered bool
	if node.LastChild != nil {
		if node.LastChild.Type == bf.Image {
			el := tr.NewElement(node.LastChild)
			err := el.Renderer.Render(w, node.LastChild, tr)
			if err != nil {
				return err
			}
		}
		if len(node.LastChild.Literal) > 0 &&
			string(node.LastChild.Literal) != string(node.LinkData.Destination) {
			textRendered = true
			el := &BaseElement{
				Token: string(node.LastChild.Literal),
				Style: ctx.style[LinkText],
			}
			err := el.Render(w, node.LastChild, tr)
			if err != nil {
				return err
			}
		}
	}

	if len(node.LinkData.Destination) > 0 {
		pre := " "
		style := ctx.style[Link]
		if !textRendered {
			pre = ""
			style.Prefix = ""
			style.Suffix = ""
		}

		el := &BaseElement{
			Token:  resolveRelativeURL(tr.BaseURL, string(node.LinkData.Destination)),
			Prefix: pre,
			Style:  style,
		}
		err := el.Render(w, node, tr)
		if err != nil {
			return err
		}
	}

	return nil
}
