package gold

import (
	"bytes"
	"fmt"
	"io"

	"github.com/yuin/goldmark/ast"
	astext "github.com/yuin/goldmark/extension/ast"
)

type ElementRenderer interface {
	Render(w io.Writer, ctx RenderContext) error
}

type ElementFinisher interface {
	Finish(w io.Writer, ctx RenderContext) error
}

type Element struct {
	Entering string
	Exiting  string
	Renderer ElementRenderer
	Finisher ElementFinisher
}

func (tr *TermRenderer) NewElement(node ast.Node, source []byte) Element {
	ctx := tr.context
	// fmt.Print(strings.Repeat("  ", ctx.blockStack.Len()), node.Type(), node.Kind())
	// defer fmt.Println()

	switch node.Kind() {
	// Document
	case ast.KindDocument:
		return Element{
			Renderer: &DocumentElement{},
			Finisher: &DocumentElement{},
		}

	// Heading
	case ast.KindHeading:
		n := node.(*ast.Heading)
		he := &HeadingElement{
			Level: n.Level,
			First: node.PreviousSibling() == nil,
		}
		return Element{
			Exiting:  "",
			Renderer: he,
			Finisher: he,
		}

	// Paragraph
	case ast.KindParagraph:
		return Element{
			Renderer: &ParagraphElement{},
			Finisher: &ParagraphElement{},
		}

	// Blockquote
	case ast.KindBlockquote:
		e := &BlockElement{
			Block:   &bytes.Buffer{},
			Style:   cascadeStyle(ctx.blockStack.Current().Style, ctx.styles.BlockQuote, true),
			Margin:  true,
			Newline: true,
		}
		return Element{
			Entering: "\n",
			Renderer: e,
			Finisher: e,
		}

	// Lists
	case ast.KindList:
		s := ctx.styles.List.StyleBlock
		if s.Indent == nil {
			var i uint
			s.Indent = &i
		}
		n := node.Parent()
		for n != nil {
			if n.Kind() == ast.KindList {
				i := ctx.styles.List.LevelIndent
				s.Indent = &i
				break
			}
			n = n.Parent()
		}

		e := &BlockElement{
			Block:   &bytes.Buffer{},
			Style:   cascadeStyle(ctx.blockStack.Current().Style, s, true),
			Margin:  true,
			Newline: true,
		}
		return Element{
			Entering: "\n",
			Renderer: e,
			Finisher: e,
		}

	case ast.KindListItem:
		var l uint
		var e uint
		l = 1
		n := node
		for n.PreviousSibling() != nil && (n.PreviousSibling().Kind() == ast.KindListItem) {
			l++
			n = n.PreviousSibling()
		}
		if node.Parent().(*ast.List).IsOrdered() {
			e = l
		}

		post := "\n"
		if node.LastChild().Kind() == ast.KindList || node.NextSibling() == nil {
			post = ""
		}

		if node.FirstChild().FirstChild().Kind() == astext.KindTaskCheckBox {
			nc := node.FirstChild().FirstChild().(*astext.TaskCheckBox)

			return Element{
				Exiting: post,
				Renderer: &CheckedItemElement{
					Checked: nc.IsChecked,
				},
			}
		}

		return Element{
			Exiting: post,
			Renderer: &ItemElement{
				Enumeration: e,
			},
		}

	// Text Elements
	case ast.KindText:
		n := node.(*ast.Text)
		s := string(n.Segment.Value(source))

		if n.HardLineBreak() || (n.SoftLineBreak()) {
			s += "\n"
		}
		return Element{
			Renderer: &BaseElement{
				Token: ctx.SanitizeHTML(s, false),
				Style: ctx.styles.Text,
			},
		}

	case ast.KindEmphasis:
		n := node.(*ast.Emphasis)
		s := string(n.Text(source))
		style := ctx.styles.Emph
		if n.Level > 1 {
			style = ctx.styles.Strong
		}

		return Element{
			Renderer: &BaseElement{
				Token: ctx.SanitizeHTML(s, false),
				Style: style,
			},
		}

	case astext.KindStrikethrough:
		n := node.(*astext.Strikethrough)
		s := string(n.Text(source))
		style := ctx.styles.Strikethrough

		return Element{
			Renderer: &BaseElement{
				Token: ctx.SanitizeHTML(s, false),
				Style: style,
			},
		}

	case ast.KindThematicBreak:
		return Element{
			Entering: "",
			Exiting:  "",
			Renderer: &BaseElement{
				Style: ctx.styles.HorizontalRule,
			},
		}

	// Links
	case ast.KindLink:
		n := node.(*ast.Link)
		text := string(n.Text(source))
		return Element{
			Renderer: &LinkElement{
				Text:    text,
				BaseURL: ctx.options.BaseURL,
				URL:     string(n.Destination),
			},
		}
	case ast.KindAutoLink:
		n := node.(*ast.AutoLink)
		text := string(n.Text(source))
		return Element{
			Renderer: &LinkElement{
				Text:    text,
				BaseURL: ctx.options.BaseURL,
				URL:     string(n.URL(source)),
			},
		}

	// Images
	case ast.KindImage:
		n := node.(*ast.Image)
		text := string(n.Text(source))
		return Element{
			Renderer: &ImageElement{
				Text:    text,
				BaseURL: ctx.options.BaseURL,
				URL:     string(n.Destination),
			},
		}

	// Code
	case ast.KindFencedCodeBlock:
		n := node.(*ast.FencedCodeBlock)
		l := n.Lines().Len()
		s := ""
		for i := 0; i < l; i++ {
			line := n.Lines().At(i)
			s += string(line.Value(source))
		}
		return Element{
			Entering: "\n",
			Renderer: &CodeBlockElement{
				Code:     s,
				Language: string(n.Language(source)),
			},
		}

	case ast.KindCodeBlock:
		n := node.(*ast.CodeBlock)
		l := n.Lines().Len()
		s := ""
		for i := 0; i < l; i++ {
			line := n.Lines().At(i)
			s += string(line.Value(source))
		}
		return Element{
			Entering: "\n",
			Renderer: &CodeBlockElement{
				Code: s,
			},
		}

	case ast.KindCodeSpan:
		// n := node.(*ast.CodeSpan)
		e := &BlockElement{
			Block: &bytes.Buffer{},
			Style: cascadeStyle(ctx.blockStack.Current().Style, ctx.styles.Code, true),
		}
		return Element{
			Renderer: e,
			Finisher: e,
		}

	// Tables
	case astext.KindTable:
		te := &TableElement{}
		return Element{
			Entering: "\n",
			Renderer: te,
			Finisher: te,
		}

	case astext.KindTableCell:
		s := ""
		n := node.FirstChild()
		for n != nil {
			s += string(n.Text(source))
			// s += string(n.LinkData.Destination)
			n = n.NextSibling()
		}

		return Element{
			Renderer: &TableCellElement{
				Text: s,
				Head: node.Parent().Kind() == astext.KindTableHeader,
			},
		}

	case astext.KindTableHeader:
		return Element{
			Finisher: &TableHeadElement{},
		}
	case astext.KindTableRow:
		return Element{
			Finisher: &TableRowElement{},
		}

	// HTML Elements
	case ast.KindHTMLBlock:
		n := node.(*ast.HTMLBlock)
		return Element{
			Renderer: &BaseElement{
				Token: ctx.SanitizeHTML(string(n.Text(source)), true) + "\n",
				Style: ctx.styles.HTMLBlock.StylePrimitive,
			},
		}
	case ast.KindRawHTML:
		n := node.(*ast.RawHTML)
		return Element{
			Renderer: &BaseElement{
				Token: ctx.SanitizeHTML(string(n.Text(source)), true) + "\n",
				Style: ctx.styles.HTMLSpan.StylePrimitive,
			},
		}

	// Definition Lists
	case astext.KindDefinitionList:
		e := &BlockElement{
			Block:   &bytes.Buffer{},
			Style:   cascadeStyle(ctx.blockStack.Current().Style, ctx.styles.DefinitionList, true),
			Margin:  true,
			Newline: true,
		}
		return Element{
			Entering: "\n",
			Renderer: e,
			Finisher: e,
		}

	case astext.KindDefinitionTerm:
		return Element{
			Renderer: &BaseElement{
				Style: ctx.styles.DefinitionTerm,
			},
		}

	case astext.KindDefinitionDescription:
		return Element{
			Renderer: &BaseElement{
				Style: ctx.styles.DefinitionDescription,
			},
		}

	// Handled by parents
	case astext.KindTaskCheckBox:
		// handled by KindListItem
		return Element{}
	case ast.KindTextBlock:
		return Element{}

	// Unknown case
	default:
		fmt.Println("Warning: unhandled element", node.Kind().String())
		return Element{}
	}
}
