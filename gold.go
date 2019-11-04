package gold

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/logrusorgru/aurora"
	bf "gopkg.in/russross/blackfriday.v2"
)

type RuleKey int

const (
	Color RuleKey = iota
	BackgroundColor
	Underline
	Bold
	Italic
	CrossedOut
	Faint
	Conceal
	Overlined
	Inverse
	Blink
)

type TermRenderer struct {
	style map[bf.NodeType]map[RuleKey]string
}

func BaseString(node *bf.Node) string {
	switch node.Type {
	case bf.Document:
		return ""
	case bf.BlockQuote:
		return fmt.Sprintf("```\n%s\n```", string(node.Literal))
	case bf.List:
		return ""
	case bf.Item:
		return fmt.Sprintf("* %s", string(node.Literal))
	case bf.Paragraph:
		if node.Prev != nil && node.Prev.Type != bf.Link {
			return "\n"
		} else {
			return ""
		}
	case bf.Heading:
		return fmt.Sprintf("%s ", strings.Repeat("#", node.HeadingData.Level))
	case bf.HorizontalRule:
		return "---\n"
	case bf.Emph:
		return fmt.Sprintf("%s\n", string(node.Literal))
	case bf.Strong:
		return fmt.Sprintf("%s\n", string(node.Literal))
	case bf.Del:
		return fmt.Sprintf("%s\n", string(node.Literal))
	case bf.Link:
		return fmt.Sprintf("[%s](%s)\n", string(node.LastChild.Literal), string(node.LinkData.Destination))
	case bf.Image:
		return fmt.Sprintf("%s\n", string(node.Literal))
	case bf.Text:
		if node.Parent != nil && node.Parent.Type != bf.Link {
			return fmt.Sprintf("%s\n", string(node.Literal))
		} else {
			return ""
		}
	case bf.HTMLBlock:
		return fmt.Sprintf("%s\n", string(node.Literal))
	case bf.CodeBlock:
		return fmt.Sprintf("%s\n", string(node.Literal))
	case bf.Softbreak:
		return fmt.Sprintf("%s\n", string(node.Literal))
	case bf.Hardbreak:
		return fmt.Sprintf("%s\n", string(node.Literal))
	case bf.Code:
		return fmt.Sprintf("%s\n", string(node.Literal))
	case bf.HTMLSpan:
		return fmt.Sprintf("%s\n", string(node.Literal))
	case bf.Table:
		return fmt.Sprintf("%s\n", string(node.Literal))
	case bf.TableCell:
		return fmt.Sprintf("%s\n", string(node.Literal))
	case bf.TableHead:
		return fmt.Sprintf("%s\n", string(node.Literal))
	case bf.TableBody:
		return fmt.Sprintf("%s\n", string(node.Literal))
	case bf.TableRow:
		return fmt.Sprintf("%s\n", string(node.Literal))
	default:
		return ""
	}

}

func (tr *TermRenderer) RenderNode(w io.Writer, node *bf.Node, entering bool) bf.WalkStatus {
	if !entering {
		return bf.GoToNext
	}
	text := BaseString(node)
	if text == "" {
		return bf.GoToNext
	}
	rules := tr.style[node.Type]
	if len(rules) == 0 {
		fmt.Fprintf(w, "%s", text)
		return bf.GoToNext
	}
	out := aurora.Reset(text)
	if r, ok := rules[Color]; ok {
		i, err := strconv.Atoi(r)
		if err == nil && i >= 0 && i <= 255 {
			out = out.Index(uint8(i))
		}
	}
	if r, ok := rules[BackgroundColor]; ok {
		i, err := strconv.Atoi(r)
		if err == nil && i >= 0 && i <= 255 {
			out = out.Index(uint8(i))
		}
	}
	if r, ok := rules[Underline]; ok {
		if r == "true" {
			out = out.Underline()
		}
	}
	if r, ok := rules[Bold]; ok {
		if r == "true" {
			out = out.Bold()
		}
	}
	if r, ok := rules[Italic]; ok {
		if r == "true" {
			out = out.Italic()
		}
	}
	if r, ok := rules[CrossedOut]; ok {
		if r == "true" {
			out = out.CrossedOut()
		}
	}
	if r, ok := rules[Overlined]; ok {
		if r == "true" {
			out = out.Overlined()
		}
	}
	if r, ok := rules[Inverse]; ok {
		if r == "true" {
			out = out.Reverse()
		}
	}
	if r, ok := rules[Blink]; ok {
		if r == "true" {
			out = out.Blink()
		}
	}
	w.Write([]byte(aurora.Sprintf("%s", out)))
	return bf.GoToNext
}

func (tr *TermRenderer) RenderHeader(w io.Writer, ast *bf.Node) {
}

func (tr *TermRenderer) RenderFooter(w io.Writer, ast *bf.Node) {
}
