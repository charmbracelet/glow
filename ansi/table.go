package ansi

import (
	"io"

	"github.com/olekukonko/tablewriter"
)

type TableElement struct {
	writer       *tablewriter.Table
	indentWriter io.Writer
	header       []string
	cell         []string
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

	var indent uint
	var margin uint
	rules := ctx.options.Styles.Table
	if rules.Indent != nil {
		indent = *rules.Indent
	}
	if rules.Margin != nil {
		margin = *rules.Margin
	}

	ctx.table.indentWriter = &IndentWriter{
		Indent: indent + margin,
		IndentFunc: func(wr io.Writer) {
			renderText(w, bs.Current().Style.StylePrimitive, " ")
		},
		Forward: &AnsiWriter{
			Forward: w,
		},
	}

	renderText(ctx.table.indentWriter, bs.Current().Style.StylePrimitive, rules.BlockPrefix)
	ctx.table.writer = tablewriter.NewWriter(ctx.table.indentWriter)
	return nil
}

func (e *TableElement) Finish(w io.Writer, ctx RenderContext) error {
	ctx.table.writer.Render()
	ctx.table.writer = nil

	rules := ctx.options.Styles.Table
	renderText(ctx.table.indentWriter, ctx.blockStack.Current().Style.StylePrimitive, rules.BlockSuffix)
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
