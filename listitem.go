package gold

import (
	"io"
	"strconv"
)

type ItemElement struct {
	Text        string
	Enumeration uint
}

func (e *ItemElement) Render(w io.Writer, ctx RenderContext) error {
	var el *BaseElement
	if e.Enumeration > 0 {

		el = &BaseElement{
			Token:  e.Text,
			Style:  ctx.style[Enumeration],
			Prefix: strconv.FormatInt(int64(e.Enumeration), 10),
		}
	} else {
		el = &BaseElement{
			Token: e.Text,
			Style: ctx.style[Item],
		}
	}

	return el.Render(w, ctx)
}
