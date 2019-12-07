package gold

import (
	"io"
)

type ImageElement struct {
	Text    string
	BaseURL string
	URL     string
	Child   ElementRenderer // FIXME
}

func (e *ImageElement) Render(w io.Writer, ctx RenderContext) error {
	if len(e.Text) > 0 {
		el := &BaseElement{
			Token: e.Text,
			Style: ctx.style[ImageText],
		}
		err := el.Render(w, ctx)
		if err != nil {
			return err
		}
	}
	if len(e.URL) > 0 {
		el := &BaseElement{
			Token:  resolveRelativeURL(e.BaseURL, string(e.URL)),
			Prefix: " ",
			Style:  ctx.style[Image],
		}
		err := el.Render(w, ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
