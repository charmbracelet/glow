package gold

import (
	"bytes"
	"io"

	bf "gopkg.in/russross/blackfriday.v2"
)

type DocumentElement struct {
}

func (e *DocumentElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	rules := tr.style[Document]
	be := BlockElement{
		Block: &bytes.Buffer{},
		Style: rules,
	}
	tr.blockStack.Push(be)

	renderText(tr.blockStack.Current().Block, rules, rules.Prefix)
	return nil
}

func (e *DocumentElement) Finish(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	var indent uint
	var margin uint
	rules := tr.style[Document]
	if rules.Indent != nil {
		indent = *rules.Indent
	}
	if rules.Margin != nil {
		margin = *rules.Margin
	}
	suffix := rules.Suffix

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
	_, err := iw.Write(tr.blockStack.Current().Block.Bytes())
	if err != nil {
		return err
	}
	renderText(iw, rules, suffix)

	tr.blockStack.Current().Block.Reset()
	tr.blockStack.Pop()

	return nil
}
