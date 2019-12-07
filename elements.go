package gold

import (
	"html"
	"io"
	"strings"

	bf "gopkg.in/russross/blackfriday.v2"
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

func (tr *TermRenderer) NewElement(node *bf.Node) Element {
	ctx := tr.context

	switch node.Type {
	case bf.Document:
		de := &DocumentElement{
			Width: uint(tr.WordWrap),
		}
		return Element{
			Renderer: de,
			Finisher: de,
		}
	case bf.BlockQuote:
		return Element{
			Entering: "\n",
			Exiting:  "\n",
			Renderer: &BaseElement{
				Token: string(node.Literal),
				Style: ctx.style[BlockQuote],
			},
		}
	case bf.List:
		le := &ListElement{
			Width:  uint(tr.WordWrap),
			Nested: node.Parent.Type == bf.Item,
		}
		return Element{
			Renderer: le,
			Finisher: le,
		}
	case bf.Item:
		var l uint
		if node.ListData.ListFlags&bf.ListTypeOrdered > 0 {
			l = 1
			n := node
			for n.Prev != nil && (n.Prev.Type == bf.Item) {
				l++
				n = n.Prev
			}
		}

		return Element{
			Renderer: &ItemElement{
				Text:        string(node.Literal),
				Enumeration: l,
			},
		}
	case bf.Paragraph:
		pe := &ParagraphElement{
			Width:      uint(tr.WordWrap),
			InsideList: node.Parent != nil && node.Parent.Type == bf.Item,
		}
		return Element{
			Renderer: pe,
			Finisher: pe,
		}
	case bf.Heading:
		return Element{
			Exiting: "\n",
			Renderer: &HeadingElement{
				Width: uint(tr.WordWrap),
				Text:  string(node.FirstChild.Literal),
				Level: node.HeadingData.Level,
				First: node.Prev == nil,
			},
		}
	case bf.HorizontalRule:
		return Element{
			Entering: "\n",
			Exiting:  "\n",
			Renderer: &BaseElement{
				Token: "---",
				Style: ctx.style[HorizontalRule],
			},
		}
	case bf.Emph:
		return Element{
			Renderer: &BaseElement{
				Token: string(node.FirstChild.Literal),
				Style: ctx.style[Emph],
			},
		}
	case bf.Strong:
		return Element{
			Renderer: &BaseElement{
				Token: string(node.FirstChild.Literal),
				Style: ctx.style[Strong],
			},
		}
	case bf.Del:
		return Element{
			Renderer: &BaseElement{
				Token: string(node.Literal),
				Style: ctx.style[Del],
			},
		}
	case bf.Link:
		var text string
		if node.LastChild != nil {
			text = string(node.LastChild.Literal)
		}
		return Element{
			Renderer: &LinkElement{
				Text:    text,
				BaseURL: tr.BaseURL,
				URL:     string(node.LinkData.Destination),
			},
		}
	case bf.Image:
		var text string
		if node.LastChild != nil {
			text = string(node.LastChild.Literal)
		}
		return Element{
			Renderer: &ImageElement{
				Text:    text,
				BaseURL: tr.BaseURL,
				URL:     string(node.LinkData.Destination),
			},
		}
	case bf.Text:
		return Element{
			Renderer: &BaseElement{
				Token: html.UnescapeString(stripper.Sanitize(string(node.Literal))),
				Style: ctx.style[Text],
			},
		}
	case bf.HTMLBlock:
		return Element{
			Renderer: &BaseElement{
				Token: html.UnescapeString(strings.TrimSpace(stripper.Sanitize(string(node.Literal)))) + "\n",
				Style: ctx.style[HTMLBlock],
			},
		}
	case bf.CodeBlock:
		return Element{
			Entering: "\n",
			Renderer: &CodeBlockElement{
				Code:     string(node.Literal),
				Language: string(node.CodeBlockData.Info),
			},
		}
	case bf.Softbreak:
		return Element{
			Exiting: "\n",
			Renderer: &BaseElement{
				Token: string(node.Literal),
				Style: ctx.style[Softbreak],
			},
		}
	case bf.Hardbreak:
		return Element{
			Exiting: "\n",
			Renderer: &BaseElement{
				Token: string(node.Literal),
				Style: ctx.style[Hardbreak],
			},
		}
	case bf.Code:
		return Element{
			Renderer: &BaseElement{
				Token: string(node.Literal),
				Style: ctx.style[Code],
			},
		}
	case bf.HTMLSpan:
		return Element{
			Renderer: &BaseElement{
				Token: html.UnescapeString(strings.TrimSpace(stripper.Sanitize(string(node.Literal)))) + "\n",
				Style: ctx.style[HTMLSpan],
			},
		}
	case bf.Table:
		te := &TableElement{}
		return Element{
			Entering: "\n",
			Renderer: te,
			Finisher: te,
		}
	case bf.TableCell:
		s := ""
		n := node.FirstChild
		for n != nil {
			s += string(n.Literal)
			s += string(n.LinkData.Destination)
			n = n.Next
		}

		return Element{
			Renderer: &TableCellElement{
				Text: s,
				Head: node.Parent.Parent.Type == bf.TableHead,
			},
		}
	case bf.TableHead:
		return Element{
			Finisher: &TableHeadElement{},
		}
	case bf.TableBody:
		return Element{}
	case bf.TableRow:
		return Element{
			Finisher: &TableRowElement{},
		}

	default:
		return Element{
			Renderer: &BaseElement{
				Token: string(node.Literal),
			},
		}
	}
}
