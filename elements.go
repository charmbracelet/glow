package gold

import (
	"html"
	"io"
	"strings"

	bf "gopkg.in/russross/blackfriday.v2"
)

type ElementRenderer interface {
	Render(w io.Writer, node *bf.Node, tr *TermRenderer) error
}

type ElementFinisher interface {
	Finish(w io.Writer, node *bf.Node, tr *TermRenderer) error
}

type Element struct {
	Entering string
	Exiting  string
	Renderer ElementRenderer
	Finisher ElementFinisher
}

func (tr *TermRenderer) NewElement(node *bf.Node) Element {
	switch node.Type {
	case bf.Document:
		de := &DocumentElement{}
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
				Style: tr.style[BlockQuote],
			},
		}
	case bf.List:
		le := &ListElement{}
		return Element{
			Renderer: le,
			Finisher: le,
		}
	case bf.Item:
		return Element{
			Renderer: &ItemElement{},
		}
	case bf.Paragraph:
		pe := &ParagraphElement{}
		return Element{
			Renderer: pe,
			Finisher: pe,
		}
	case bf.Heading:
		return Element{
			Exiting:  "\n",
			Renderer: &HeadingElement{},
		}
	case bf.HorizontalRule:
		return Element{
			Entering: "\n",
			Exiting:  "\n",
			Renderer: &BaseElement{
				Token: "---",
				Style: tr.style[HorizontalRule],
			},
		}
	case bf.Emph:
		return Element{
			Renderer: &BaseElement{
				Token: string(node.FirstChild.Literal),
				Style: tr.style[Emph],
			},
		}
	case bf.Strong:
		return Element{
			Renderer: &BaseElement{
				Token: string(node.FirstChild.Literal),
				Style: tr.style[Strong],
			},
		}
	case bf.Del:
		return Element{
			Renderer: &BaseElement{
				Token: string(node.Literal),
				Style: tr.style[Del],
			},
		}
	case bf.Link:
		return Element{
			Renderer: &LinkElement{},
		}
	case bf.Image:
		return Element{
			Renderer: &ImageElement{},
		}
	case bf.Text:
		return Element{
			Renderer: &BaseElement{
				Token: html.UnescapeString(stripper.Sanitize(string(node.Literal))),
				Style: tr.style[Text],
			},
		}
	case bf.HTMLBlock:
		return Element{
			Renderer: &BaseElement{
				Token: html.UnescapeString(strings.TrimSpace(stripper.Sanitize(string(node.Literal)))) + "\n",
				Style: tr.style[HTMLBlock],
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
				Style: tr.style[Softbreak],
			},
		}
	case bf.Hardbreak:
		return Element{
			Exiting: "\n",
			Renderer: &BaseElement{
				Token: string(node.Literal),
				Style: tr.style[Hardbreak],
			},
		}
	case bf.Code:
		return Element{
			Renderer: &BaseElement{
				Token: string(node.Literal),
				Style: tr.style[Code],
			},
		}
	case bf.HTMLSpan:
		return Element{
			Renderer: &BaseElement{
				Token: html.UnescapeString(strings.TrimSpace(stripper.Sanitize(string(node.Literal)))) + "\n",
				Style: tr.style[HTMLSpan],
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
		return Element{
			Renderer: &TableCellElement{},
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
