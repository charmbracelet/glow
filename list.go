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
	ctx := tr.context
	rules := ctx.style[List]
	if node.Parent.Type != bf.Item {
		_, _ = w.Write([]byte("\n"))
	}

	be := BlockElement{
		Block: &bytes.Buffer{},
		Style: cascadeStyle(ctx.blockStack.Current().Style, rules, true),
	}
	ctx.blockStack.Push(be)

	return nil
}

func (e *ListElement) Finish(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	ctx := tr.context
	bs := ctx.blockStack
	var indent uint
	var margin uint
	rules := bs.Current().Style
	if rules.Indent != nil {
		indent = *rules.Indent
	}
	if rules.Margin != nil {
		margin = *rules.Margin
	}
	suffix := rules.Suffix
	renderText(bs.Current().Block, rules, suffix)

	pw := &PaddingWriter{
		Padding: uint(tr.WordWrap - int(bs.Indent()) - int(bs.Margin()*2)),
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
			renderText(w, bs.Parent().Style, " ")
		},
		Forward: &AnsiWriter{
			Forward: pw,
		},
	}

	_, err := iw.Write(reflow.Bytes(bs.Current().Block.Bytes(),
		tr.WordWrap-int(bs.Indent())-int(bs.Margin())*2))
	if err != nil {
		return err
	}

	bs.Current().Block.Reset()
	bs.Pop()
	return nil
}
