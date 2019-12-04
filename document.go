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
		renderText(w, rules, rules.Prefix)
	}
	return nil
}

func (e *DocumentElement) Finish(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	var indent uint
	var margin uint
	var suffix string
	rules := tr.style[Document]
	if rules != nil {
		indent = rules.Indent
		margin = rules.Margin
		suffix = rules.Suffix
	}
	pw := &PaddingWriter{
		Padding: uint(tr.WordWrap) - margin,
		PadFunc: func(wr io.Writer) {
			renderText(w, rules, " ")
		},
		Forward: &AnsiWriter{
			Forward: w,
		},
	}
	iw := &IndentWriter{
		Indent: indent + margin,
		Forward: &AnsiWriter{
			Forward: pw,
		},
	}
	_, err := iw.Write(tr.document.Bytes())
	if err != nil {
		return err
	}
	renderText(iw, rules, suffix)

	tr.document.Reset()
	tr.blockStyle.Pop()

	return nil
}
