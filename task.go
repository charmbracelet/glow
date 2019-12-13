package gold

import (
	"io"
)

type TaskElement struct {
	Checked bool
}

func (e *TaskElement) Render(w io.Writer, ctx RenderContext) error {
	var el *BaseElement

	pre := ctx.styles.Task.Unticked
	if e.Checked {
		pre = ctx.styles.Task.Ticked
	}

	el = &BaseElement{
		Prefix: pre,
		Style:  ctx.styles.Task.StylePrimitive,
	}

	return el.Render(w, ctx)
}
