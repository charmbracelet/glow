package gold

import (
	"io"
	"strconv"
)

type ItemElement struct {
	Enumeration uint
}

func (e *ItemElement) Render(w io.Writer, ctx RenderContext) error {
	var el *BaseElement
	if e.Enumeration > 0 {
		el = &BaseElement{
			Style:  ctx.styles.Enumeration,
			Prefix: strconv.FormatInt(int64(e.Enumeration), 10),
		}
	} else {
		el = &BaseElement{
			Style: ctx.styles.Item,
		}
	}

	return el.Render(w, ctx)
}
