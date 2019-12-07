package gold

import (
	"io"
)

type LinkElement struct {
	Text    string
	BaseURL string
	URL     string
	Child   ElementRenderer // FIXME
}

func (e *LinkElement) Render(w io.Writer, ctx RenderContext) error {
	var textRendered bool
	if len(e.Text) > 0 &&
		e.Text != e.URL {
		textRendered = true

		el := &BaseElement{
			Token: e.Text,
			Style: ctx.style[LinkText],
		}
		err := el.Render(w, ctx)
		if err != nil {
			return err
		}
	}

	/*
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
	*/

	if len(e.URL) > 0 {
		pre := " "
		style := ctx.style[Link]
		if !textRendered {
			pre = ""
			style.Prefix = ""
			style.Suffix = ""
		}

		el := &BaseElement{
			Token:  resolveRelativeURL(e.BaseURL, e.URL),
			Prefix: pre,
			Style:  style,
		}
		err := el.Render(w, ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
