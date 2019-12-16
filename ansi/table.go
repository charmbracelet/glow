package ansi

import (
	"io"

	"github.com/muesli/reflow/ansi"
	"github.com/muesli/reflow/indent"
	"github.com/olekukonko/tablewriter"
)

type TableElement struct {
	writer      *tablewriter.Table
	styleWriter *StyleWriter
	header      []string
	cell        []string
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
	bs := ctx.blockStack

	var indentation uint
	var margin uint
	rules := ctx.options.Styles.Table
	if rules.Indent != nil {
		indentation = *rules.Indent
	}
	if rules.Margin != nil {
		margin = *rules.Margin
	}

	iw := &indent.Writer{
		Indent: indentation + margin,
		IndentFunc: func(wr io.Writer) {
			renderText(w, bs.Current().Style.StylePrimitive, " ")
		},
		Forward: &ansi.Writer{
			Forward: w,
		},
	}

	ctx.table.styleWriter = NewStyleWriter(ctx, iw, rules.StylePrimitive)

	renderText(w, bs.Current().Style.StylePrimitive, rules.BlockPrefix)
	renderText(ctx.table.styleWriter, rules.StylePrimitive, rules.Prefix)
	ctx.table.writer = tablewriter.NewWriter(ctx.table.styleWriter)
	return nil
}

func (e *TableElement) Finish(w io.Writer, ctx RenderContext) error {
	rules := ctx.options.Styles.Table

	ctx.table.writer.SetBorders(tablewriter.Border{Left: false, Top: false, Right: false, Bottom: false})
	if rules.CenterSeparator != nil {
		ctx.table.writer.SetCenterSeparator(*rules.CenterSeparator)
	}
	if rules.ColumnSeparator != nil {
		ctx.table.writer.SetColumnSeparator(*rules.ColumnSeparator)
	}
	if rules.RowSeparator != nil {
		ctx.table.writer.SetRowSeparator(*rules.RowSeparator)
	}

	ctx.table.writer.Render()
	ctx.table.writer = nil

	renderText(ctx.table.styleWriter, rules.StylePrimitive, rules.Suffix)
	renderText(ctx.table.styleWriter, ctx.blockStack.Current().Style.StylePrimitive, rules.BlockSuffix)
	return ctx.table.styleWriter.Close()
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
