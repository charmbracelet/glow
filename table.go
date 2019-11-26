package gold

import (
	"io"

	"github.com/olekukonko/tablewriter"
	bf "gopkg.in/russross/blackfriday.v2"
)

type TableElement struct {
}

type TableRowElement struct {
}

type TableHeadElement struct {
}

type TableCellElement struct {
}

func (e *TableElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	tr.tableData.table = tablewriter.NewWriter(w)
	return nil
}

func (e *TableElement) Finish(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	tr.tableData.table.Render()
	tr.tableData = TableData{}
	return nil
}

func (e *TableRowElement) Finish(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	tr.tableData.table.Append(tr.tableData.tableCell)
	tr.tableData.tableCell = []string{}
	return nil
}

func (e *TableHeadElement) Finish(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	tr.tableData.table.SetHeader(tr.tableData.tableHeader)
	tr.tableData.tableHeader = []string{}
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
		tr.tableData.tableHeader = append(tr.tableData.tableHeader, s)
	} else {
		tr.tableData.tableCell = append(tr.tableData.tableCell, s)
	}

	return nil
}
