package ansi

import (
	"io"

	"github.com/alecthomas/chroma/quick"
)

type CodeBlockElement struct {
	Code     string
	Language string
}

func (e *CodeBlockElement) Render(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack

	var indent uint
	var margin uint
	rules := ctx.options.Styles.CodeBlock
	if rules.Indent != nil {
		indent = *rules.Indent
	}
	if rules.Margin != nil {
		margin = *rules.Margin
	}
	theme := rules.Theme

	iw := &IndentWriter{
		Indent: indent + margin,
		IndentFunc: func(wr io.Writer) {
			renderText(w, bs.Current().Style.StylePrimitive, " ")
		},
		Forward: &AnsiWriter{
			Forward: w,
		},
	}

	if len(theme) > 0 {
		renderText(iw, bs.Current().Style.StylePrimitive, rules.BlockPrefix)
		err := quick.Highlight(iw, e.Code, e.Language, "terminal16m", theme)
		renderText(iw, bs.Current().Style.StylePrimitive, rules.BlockSuffix)
		return err
	}

	// fallback rendering
	el := &BaseElement{
		Token: e.Code,
		Style: rules.StylePrimitive,
	}

	return el.Render(iw, ctx)
}
