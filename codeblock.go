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
	rules := tr.style[CodeBlock]
	if rules != nil {
		indent = rules.Indent
		theme = rules.Theme
	}

	iw := &IndentWriter{
		Indent: indent,
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
		Style: CodeBlock,
	}

	return el.Render(iw, node, tr)
}
