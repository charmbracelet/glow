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

type Element struct {
	Entering string
	Exiting  string
	Renderer ElementRenderer
}

func (tr *TermRenderer) NewElement(node *bf.Node) Element {
	switch node.Type {
	case bf.Document:
		return Element{
			Renderer: &BaseElement{
				Style: Document,
			},
		}
	case bf.BlockQuote:
		return Element{
			Entering: "\n",
			Exiting:  "\n",
			Renderer: &BaseElement{
				Token: string(node.Literal),
				Style: BlockQuote,
			},
		}
	case bf.List:
		return Element{
			Renderer: &ListElement{},
		}
	case bf.Item:
		return Element{
			Renderer: &ItemElement{},
		}
	case bf.Paragraph:
		return Element{
			Exiting:  "\n",
			Renderer: &ParagraphElement{},
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
				Style: HorizontalRule,
			},
		}
	case bf.Emph:
		return Element{
			Renderer: &BaseElement{
				Token: string(node.FirstChild.Literal),
				Style: Emph,
			},
		}
	case bf.Strong:
		return Element{
			Renderer: &BaseElement{
				Token: string(node.FirstChild.Literal),
				Style: Strong,
			},
		}
	case bf.Del:
		return Element{
			Renderer: &BaseElement{
				Token: string(node.Literal),
				Style: Del,
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
				Style: Text,
			},
		}
	case bf.HTMLBlock:
		return Element{
			Renderer: &BaseElement{
				Token: html.UnescapeString(strings.TrimSpace(stripper.Sanitize(string(node.Literal)))) + "\n",
				Style: HTMLBlock,
			},
		}
	case bf.CodeBlock:
		return Element{
			Entering: "\n",
			Exiting:  "\n",
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
				Style: Softbreak,
			},
		}
	case bf.Hardbreak:
		return Element{
			Exiting: "\n",
			Renderer: &BaseElement{
				Token: string(node.Literal),
				Style: Hardbreak,
			},
		}
	case bf.Code:
		return Element{
			Renderer: &BaseElement{
				Token: string(node.Literal),
				Style: Code,
			},
		}
	case bf.HTMLSpan:
		return Element{
			Renderer: &BaseElement{
				Token: html.UnescapeString(strings.TrimSpace(stripper.Sanitize(string(node.Literal)))) + "\n",
				Style: HTMLSpan,
			},
		}
	case bf.Table:
		return Element{
			Entering: "\n",
			Exiting:  "\n",
		}
	case bf.TableCell:
		return Element{}
	case bf.TableHead:
		return Element{}
	case bf.TableBody:
		return Element{}
	case bf.TableRow:
		return Element{
			Entering: "\n",
			Exiting:  "\n",
		}

	default:
		return Element{
			Renderer: &BaseElement{
				Token: string(node.Literal),
			},
		}
	}
}
