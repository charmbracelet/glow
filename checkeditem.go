package gold

import (
	"io"
)

type CheckedItemElement struct {
	Checked bool
}

func (e *CheckedItemElement) Render(w io.Writer, ctx RenderContext) error {
	var el *BaseElement

	pre := "✗ "
	if e.Checked {
		pre = "✓ "
	}

	el = &BaseElement{
		Prefix: pre,
		Style:  ctx.style[CheckedItem],
	}

	return el.Render(w, ctx)
}
