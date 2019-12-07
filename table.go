package gold

import (
	"io"

	"github.com/olekukonko/tablewriter"
	bf "gopkg.in/russross/blackfriday.v2"
)

type TableElement struct {
	writer *tablewriter.Table
	header []string
	cell   []string
}

type TableRowElement struct {
}

type TableHeadElement struct {
}

type TableCellElement struct {
}

func (e *TableElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	ctx := tr.context

	var indent uint
	var margin uint
	rules := ctx.style[Table]
	if rules.Indent != nil {
		indent = *rules.Indent
	}
	if rules.Margin != nil {
		margin = *rules.Margin
	}

	iw := &IndentWriter{
		Indent: indent + margin,
		IndentFunc: func(wr io.Writer) {
			renderText(w, ctx.blockStack.Parent().Style, " ")
		},
		Forward: &AnsiWriter{
			Forward: w,
		},
	}

	ctx.table.writer = tablewriter.NewWriter(iw)
	return nil
}

func (e *TableElement) Finish(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	ctx := tr.context
	ctx.table.writer.Render()
	ctx.table.writer = nil
	return nil
}

func (e *TableRowElement) Finish(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	ctx := tr.context
	ctx.table.writer.Append(ctx.table.cell)
	ctx.table.cell = []string{}
	return nil
}

func (e *TableHeadElement) Finish(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	ctx := tr.context
	ctx.table.writer.SetHeader(ctx.table.header)
	ctx.table.header = []string{}
	return nil
}

func (e *TableCellElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	ctx := tr.context

	s := ""
	n := node.FirstChild
	for n != nil {
		s += string(n.Literal)
		s += string(n.LinkData.Destination)
		n = n.Next
	}

	if node.Parent.Parent.Type == bf.TableHead {
		ctx.table.header = append(ctx.table.header, s)
	} else {
		ctx.table.cell = append(ctx.table.cell, s)
	}

	return nil
}
