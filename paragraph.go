package gold

import (
	"bytes"
	"io"

	"github.com/muesli/reflow"
	bf "gopkg.in/russross/blackfriday.v2"
)

type ParagraphElement struct {
}

func (e *ParagraphElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	tr.paragraph = &bytes.Buffer{}

	rules := tr.style[Paragraph]
	tr.blockStyle.Push(rules)

	if node.Prev == nil || (node.Parent != nil && node.Parent.Type == bf.Item) {
		// list item
	} else {
		_, _ = w.Write([]byte("\n"))
	}

	if rules != nil {
		renderText(w, tr.blockStyle.Current(), rules.Prefix)
	}
	return nil
}

func (e *ParagraphElement) Finish(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	var indent uint
	var margin uint
	var suffix string
	rules := tr.blockStyle.Current()
	if rules != nil {
		indent = rules.Indent
		margin = rules.Margin
		suffix = rules.Suffix

		if node.Prev == nil || (node.Parent != nil && node.Parent.Type == bf.Item) {
			// remove indent & margin for list items
			indent = 0
			margin = 0
		}
	}
	renderText(tr.paragraph, rules, suffix)

	pw := &PaddingWriter{
		Padding: uint(tr.WordWrap - int(tr.blockStyle.Indent()) - int(tr.blockStyle.Margin()*2)),
		PadFunc: func(wr io.Writer) {
			renderText(w, rules, " ")
		},
		Forward: &AnsiWriter{
			Forward: w,
		},
	}
	iw := &IndentWriter{
		Indent: indent + margin,
		IndentFunc: func(wr io.Writer) {
			renderText(w, tr.blockStyle.Parent(), " ")
		},
		Forward: &AnsiWriter{
			Forward: pw,
		},
	}

	_, err := iw.Write(reflow.ReflowBytes(tr.paragraph.Bytes(), tr.WordWrap-int(tr.blockStyle.Indent())-int(tr.blockStyle.Margin())*2))
	if err != nil {
		return err
	}
	_, _ = pw.Write([]byte("\n"))

	tr.paragraph.Reset()
	tr.paragraph = nil

	if rules != nil {
		tr.blockStyle.Pop()
	}
	return nil
}
