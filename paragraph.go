package gold

import (
	"bytes"
	"io"
	"strings"

	"github.com/muesli/reflow"
	bf "gopkg.in/russross/blackfriday.v2"
)

type ParagraphElement struct {
}

func (e *ParagraphElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	var rules *ElementStyle
	if node.Parent != nil && node.Parent.Type == bf.Item {
		// list item
		rules = tr.style[List]
	} else {
		rules = tr.style[Paragraph]
		_, _ = w.Write([]byte("\n"))
		be := BlockElement{
			Block: &bytes.Buffer{},
			Style: cascadeStyle(tr.blockStack.Current().Style, rules, true),
		}
		tr.blockStack.Push(be)
	}

	if rules != nil {
		renderText(w, tr.blockStack.Current().Style, rules.Prefix)
	}
	return nil
}

func (e *ParagraphElement) Finish(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	var indent uint
	var margin uint
	var suffix string
	rules := tr.blockStack.Current().Style
	if node.Parent != nil && node.Parent.Type == bf.Item {
		// remove indent & margin for list items
		rules = tr.blockStack.Current().Style
	}

	if rules != nil {
		if rules.Indent != nil {
			indent = *rules.Indent
		}
		if rules.Margin != nil {
			margin = *rules.Margin
		}
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

	if len(strings.TrimSpace(tr.blockStack.Current().Block.String())) > 0 {
		_, err := iw.Write(reflow.ReflowBytes(tr.blockStack.Current().Block.Bytes(), tr.WordWrap-int(tr.blockStack.Indent())-int(tr.blockStack.Margin())*2))
		if err != nil {
			return err
		}
		_, _ = pw.Write([]byte("\n"))
	}

	tr.blockStack.Current().Block.Reset()
	if node.Parent != nil && node.Parent.Type == bf.Item {
	} else {
		tr.blockStack.Pop()
	}
	return nil
}
