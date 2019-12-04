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
	var indent uint
	rules := tr.style[Table]
	if rules != nil {
		indent = rules.Indent
	}
	iw := &IndentWriter{
		Indent: indent,
		Forward: &AnsiWriter{
			Forward: w,
		},
	}

	tr.table.writer = tablewriter.NewWriter(iw)
	return nil
}

func (e *TableElement) Finish(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	tr.table.writer.Render()
	tr.table.writer = nil
	return nil
}

func (e *TableRowElement) Finish(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	tr.table.writer.Append(tr.table.cell)
	tr.table.cell = []string{}
	return nil
}

func (e *TableHeadElement) Finish(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	tr.table.writer.SetHeader(tr.table.header)
	tr.table.header = []string{}
	return nil
}

func (e *TableCellElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	s := ""
	n := node.FirstChild
	for n != nil {
		s += string(n.Literal)
		s += string(n.LinkData.Destination)
		n = n.Next
	}

	if node.Parent.Parent.Type == bf.TableHead {
		tr.table.header = append(tr.table.header, s)
	} else {
		tr.table.cell = append(tr.table.cell, s)
	}

	return nil
}
