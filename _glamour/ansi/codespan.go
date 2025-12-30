package ansi

import "io"

// A CodeSpanElement is used to render codespan.
type CodeSpanElement struct {
	Text  string
	Style StylePrimitive
}

// Render renders a CodeSpanElement.
func (e *CodeSpanElement) Render(w io.Writer, ctx RenderContext) error {
	renderText(w, ctx.options.ColorProfile, e.Style, e.Style.Prefix+e.Text+e.Style.Suffix)
	return nil
}
