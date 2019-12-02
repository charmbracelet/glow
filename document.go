package gold

import (
	"io"

	bf "gopkg.in/russross/blackfriday.v2"
)

type DocumentElement struct {
}

func (e *DocumentElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	rules := tr.style[Document]
	if rules != nil {
		tr.blockStyle.Push(rules)
		if rules.Prefix != "" {
			renderText(w, rules, rules.Prefix)
		}
	}
	return nil
}

func (e *DocumentElement) Finish(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	var indent uint
	suffix := ""
	rules := tr.style[Document]
	if rules != nil {
		indent = rules.Indent
		suffix = rules.Suffix
	}

	iw := &IndentWriter{
		Indent:  indent,
		Forward: w,
	}

	_, err := iw.Write(tr.document.Bytes())
	if err != nil {
		return err
	}
	tr.document.Reset()
	tr.blockStyle.Pop()

	if suffix != "" {
		renderText(iw, rules, suffix)
	}
	return nil
}
