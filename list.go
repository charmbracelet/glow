package gold

import (
	"bytes"
	"io"

	"github.com/muesli/reflow"
	bf "gopkg.in/russross/blackfriday.v2"
)

type ListElement struct {
}

func (e *ListElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	rules := tr.style[List]
	if node.Parent.Type != bf.Item {
		_, _ = w.Write([]byte("\n"))
	}

	be := BlockElement{
		Block: &bytes.Buffer{},
		Style: cascadeStyle(tr.blockStack.Current().Style, rules),
	}
	tr.blockStack.Push(be)

	if rules != nil {
		renderText(w, tr.blockStack.Current().Style, rules.Prefix)
	}
	return nil
}

func (e *ListElement) Finish(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	var indent uint
	var margin uint
	var suffix string
	rules := tr.blockStack.Current().Style
	if rules != nil {
		indent = rules.Indent
		margin = rules.Margin
		suffix = rules.Suffix
	}
	renderText(tr.blockStack.Current().Block, rules, suffix)

	pw := &PaddingWriter{
		Padding: uint(tr.WordWrap - int(tr.blockStack.Indent()) - int(tr.blockStack.Margin()*2)),
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
			renderText(w, tr.blockStack.Parent().Style, " ")
		},
		Forward: &AnsiWriter{
			Forward: pw,
		},
	}

	_, err := iw.Write(reflow.ReflowBytes(tr.blockStack.Current().Block.Bytes(), tr.WordWrap-int(tr.blockStack.Indent())-int(tr.blockStack.Margin())*2))
	if err != nil {
		return err
	}

	tr.blockStack.Current().Block.Reset()
	tr.blockStack.Pop()
	return nil
}
