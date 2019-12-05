package gold

import (
	"io"

	"github.com/alecthomas/chroma/quick"
	bf "gopkg.in/russross/blackfriday.v2"
)

type CodeBlockElement struct {
	Code     string
	Language string
}

func (e *CodeBlockElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	var theme string
	var indent uint
	var margin uint
	rules := tr.style[CodeBlock]
	if rules != nil {
		if rules.Indent != nil {
			indent = *rules.Indent
		}
		if rules.Margin != nil {
			margin = *rules.Margin
		}
		theme = rules.Theme
	}

	iw := &IndentWriter{
		Indent: indent + margin,
		IndentFunc: func(wr io.Writer) {
			renderText(w, tr.blockStack.Parent().Style, " ")
		},
		Forward: &AnsiWriter{
			Forward: w,
		},
	}

	if len(theme) > 0 {
		return quick.Highlight(iw, e.Code, e.Language, "terminal16m", theme)
	}

	// fallback rendering
	el := &BaseElement{
		Token: string(e.Code),
		Style: rules,
	}

	return el.Render(iw, node, tr)
}
