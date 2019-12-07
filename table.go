package gold

import (
	"io"

	"github.com/olekukonko/tablewriter"
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
	Text string
	Head bool
}

func (e *TableElement) Render(w io.Writer, ctx RenderContext) error {
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

func (e *TableElement) Finish(w io.Writer, ctx RenderContext) error {
	ctx.table.writer.Render()
	ctx.table.writer = nil
	return nil
}

func (e *TableRowElement) Finish(w io.Writer, ctx RenderContext) error {
	ctx.table.writer.Append(ctx.table.cell)
	ctx.table.cell = []string{}
	return nil
}

func (e *TableHeadElement) Finish(w io.Writer, ctx RenderContext) error {
	ctx.table.writer.SetHeader(ctx.table.header)
	ctx.table.header = []string{}
	return nil
}

func (e *TableCellElement) Render(w io.Writer, ctx RenderContext) error {
	if e.Head {
		ctx.table.header = append(ctx.table.header, e.Text)
	} else {
		ctx.table.cell = append(ctx.table.cell, e.Text)
	}

	return nil
}
