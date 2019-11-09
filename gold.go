package gold

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
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

type ElementStyle struct {
	Color           string `json:"color"`
	BackgroundColor string `json:"background_color"`
	Underline       bool   `json:"underline"`
	Bold            bool   `json:"bold"`
	Italic          bool   `json:"italic"`
	CrossedOut      bool   `json:"crossed_out"`
	Faint           bool   `json:"faint"`
	Conceal         bool   `json:"conceal"`
	Overlined       bool   `json:"overlined"`
	Inverse         bool   `json:"inverse"`
	Blink           bool   `json:"blink"`
}

type Element struct {
	BaseString string
}

type TermRenderer struct {
	style map[bf.NodeType]*ElementStyle
}

func NewElement(node *bf.Node) Element {
	switch node.Type {
	case bf.Document:
		return Element{""}
	case bf.BlockQuote:
		return Element{fmt.Sprintf("```\n%s\n```", string(node.Literal))}
	case bf.List:
		return Element{""}
	case bf.Item:
		return Element{fmt.Sprintf("* %s", string(node.Literal))}
	case bf.Paragraph:
		if node.Prev != nil && node.Prev.Type != bf.Link {
			return Element{"\n"}
		} else {
			return Element{""}
		}
	case bf.Heading:
		return Element{fmt.Sprintf("%s ", strings.Repeat("#", node.HeadingData.Level))}
	case bf.HorizontalRule:
		return Element{"---\n"}
	case bf.Emph:
		return Element{fmt.Sprintf("%s\n", string(node.Literal))}
	case bf.Strong:
		return Element{fmt.Sprintf("%s\n", string(node.Literal))}
	case bf.Del:
		return Element{fmt.Sprintf("%s\n", string(node.Literal))}
	case bf.Link:
		return Element{fmt.Sprintf("[%s](%s)\n", string(node.LastChild.Literal), string(node.LinkData.Destination))}
	case bf.Image:
		return Element{fmt.Sprintf("%s\n", string(node.Literal))}
	case bf.Text:
		if node.Parent != nil && node.Parent.Type != bf.Link {
			return Element{fmt.Sprintf("%s\n", string(node.Literal))}
		} else {
			return Element{""}
		}
	case bf.HTMLBlock:
		return Element{fmt.Sprintf("%s\n", string(node.Literal))}
	case bf.CodeBlock:
		return Element{fmt.Sprintf("%s\n", string(node.Literal))}
	case bf.Softbreak:
		return Element{fmt.Sprintf("%s\n", string(node.Literal))}
	case bf.Hardbreak:
		return Element{fmt.Sprintf("%s\n", string(node.Literal))}
	case bf.Code:
		return Element{fmt.Sprintf("%s\n", string(node.Literal))}
	case bf.HTMLSpan:
		return Element{fmt.Sprintf("%s\n", string(node.Literal))}
	case bf.Table:
		return Element{fmt.Sprintf("%s\n", string(node.Literal))}
	case bf.TableCell:
		return Element{fmt.Sprintf("%s\n", string(node.Literal))}
	case bf.TableHead:
		return Element{fmt.Sprintf("%s\n", string(node.Literal))}
	case bf.TableBody:
		return Element{fmt.Sprintf("%s\n", string(node.Literal))}
	case bf.TableRow:
		return Element{fmt.Sprintf("%s\n", string(node.Literal))}
	default:
		return Element{""}
	}

}

func NewTermRenderer(stylePath string) (*TermRenderer, error) {
	if stylePath == "" {
		return NewTermRendererFromBytes([]byte("{}"))
	}
	f, err := os.Open(stylePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, _ := ioutil.ReadAll(f)
	return NewTermRendererFromBytes(b)
}

func NewTermRendererFromBytes(b []byte) (*TermRenderer, error) {
	e := make(map[string]*ElementStyle, 0)
	err := json.Unmarshal(b, &e)
	if err != nil {
		return nil, err
	}
	tr := &TermRenderer{}
	tr.style = make(map[bf.NodeType]*ElementStyle)

	for k, v := range e {
		t, err := keyToType(k)
		if err != nil {
			fmt.Println(err)
			continue
		}
		tr.style[t] = v
	}
	return tr, nil
}

func (tr *TermRenderer) RenderNode(w io.Writer, node *bf.Node, entering bool) bf.WalkStatus {
	if !entering {
		return bf.GoToNext
	}
	el := NewElement(node)
	if el.BaseString == "" {
		return bf.GoToNext
	}
	rules := tr.style[node.Type]
	if rules == nil {
		fmt.Fprintf(w, "%s", el.BaseString)
		return bf.GoToNext
	}
	out := aurora.Reset(el.BaseString)
	if rules.Color != "" {
		i, err := strconv.Atoi(rules.Color)
		if err == nil && i >= 0 && i <= 255 {
			out = out.Index(uint8(i))
		}
	}
	if rules.BackgroundColor != "" {
		i, err := strconv.Atoi(rules.BackgroundColor)
		if err == nil && i >= 0 && i <= 255 {
			out = out.Index(uint8(i))
		}
	}
	if rules.Underline {
		out = out.Underline()
	}
	if rules.Bold {
		out = out.Bold()
	}
	if rules.Italic {
		out = out.Italic()
	}
	if rules.CrossedOut {
		out = out.CrossedOut()
	}
	if rules.Overlined {
		out = out.Overlined()
	}
	if rules.Inverse {
		out = out.Reverse()
	}
	if rules.Blink {
		out = out.Blink()
	}
	w.Write([]byte(aurora.Sprintf("%s", out)))
	return bf.GoToNext
}

func (tr *TermRenderer) RenderHeader(w io.Writer, ast *bf.Node) {
}

func (tr *TermRenderer) RenderFooter(w io.Writer, ast *bf.Node) {
}

func keyToType(key string) (bf.NodeType, error) {
	switch key {
	case "document":
		return bf.Document, nil
	case "block_quote":
		return bf.BlockQuote, nil
	case "list":
		return bf.List, nil
	case "item":
		return bf.Item, nil
	case "paragraph":
		return bf.Paragraph, nil
	case "heading":
		return bf.Heading, nil
	case "hr":
		return bf.HorizontalRule, nil
	case "emph":
		return bf.Emph, nil
	case "strong":
		return bf.Strong, nil
	case "del":
		return bf.Del, nil
	case "link":
		return bf.Link, nil
	case "image":
		return bf.Image, nil
	case "text":
		return bf.Text, nil
	case "html_block":
		return bf.HTMLBlock, nil
	case "code_block":
		return bf.CodeBlock, nil
	case "softbreak":
		return bf.Softbreak, nil
	case "hardbreak":
		return bf.Hardbreak, nil
	case "code":
		return bf.Code, nil
	case "html_span":
		return bf.HTMLSpan, nil
	case "table":
		return bf.Table, nil
	case "table_cel":
		return bf.TableCell, nil
	case "table_head":
		return bf.TableHead, nil
	case "table_body":
		return bf.TableBody, nil
	case "table_row":
		return bf.TableRow, nil
	default:
		return 0, fmt.Errorf("Invalid element type: %s", key)
	}
}
