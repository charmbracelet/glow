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

type Fragment struct {
	Token string
	Pre   string
	Post  string
	Style StyleType
}

type Element struct {
	Pre       string
	Post      string
	Fragments []Fragment
}

type TermRenderer struct {
	style map[StyleType]*ElementStyle
}

func Render(in string, stylePath string) ([]byte, error) {
	return RenderBytes([]byte(in), stylePath)
}

func RenderBytes(in []byte, stylePath string) ([]byte, error) {
	r, err := NewTermRenderer(stylePath)
	if err != nil {
		return nil, err
	}
	return r.RenderBytes(in), nil
}

func NewPlainTermRenderer() *TermRenderer {
	return &TermRenderer{}
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
	e := make(map[string]*ElementStyle)
	err := json.Unmarshal(b, &e)
	if err != nil {
		return nil, err
	}
	tr := &TermRenderer{}
	tr.style = make(map[StyleType]*ElementStyle)

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

func NewElement(node *bf.Node) Element {
	switch node.Type {
	case bf.Document:
		return Element{
			Pre:  "",
			Post: "",
			Fragments: []Fragment{{
				Token: "",
				Style: Document,
			}},
		}
	case bf.BlockQuote:
		return Element{
			Pre:  "\n",
			Post: "\n",
			Fragments: []Fragment{{
				Token: string(node.Literal),
				Style: BlockQuote,
			}},
		}
	case bf.List:
		return Element{
			Pre:  "\n",
			Post: "",
			Fragments: []Fragment{{
				Token: string(node.Literal),
				Style: List,
			}},
		}
	case bf.Item:
		return Element{
			Pre:  "• ",
			Post: "",
			Fragments: []Fragment{{
				Token: string(node.Literal),
				Style: Item,
			}},
		}
	case bf.Paragraph:
		pre := "\n"
		if node.Prev == nil || (node.Parent != nil && node.Parent.Type == bf.Item) {
			pre = ""
		}

		return Element{
			Pre:  pre,
			Post: "\n",
			Fragments: []Fragment{{
				Token: string(node.Literal),
				Style: Paragraph,
			}},
		}
	case bf.Heading:
		var pre string
		if node.Prev != nil {
			pre = "\n"
		}

		return Element{
			Pre:  pre,
			Post: "\n",
			Fragments: []Fragment{{
				Token: fmt.Sprintf("%s %s", strings.Repeat("#", node.HeadingData.Level), node.FirstChild.Literal),
				Style: Heading,
			}},
		}
	case bf.HorizontalRule:
		return Element{
			Pre:  "\n",
			Post: "\n",
			Fragments: []Fragment{{
				Token: "---",
				Style: HorizontalRule,
			}},
		}
	case bf.Emph:
		return Element{
			Pre:  "",
			Post: "",
			Fragments: []Fragment{{
				Token: string(node.FirstChild.Literal),
				Style: Emph,
			}},
		}
	case bf.Strong:
		return Element{
			Pre:  "",
			Post: "",
			Fragments: []Fragment{{
				Token: string(node.FirstChild.Literal),
				Style: Strong,
			}},
		}
	case bf.Del:
		return Element{
			Pre:  "",
			Post: "",
			Fragments: []Fragment{{
				Token: string(node.Literal),
				Style: Del,
			}},
		}

	case bf.Link:
		f := []Fragment{}

		if node.LastChild != nil {
			if node.LastChild.Type == bf.Image {
				el := NewElement(node.LastChild)
				f = el.Fragments
			}
			if len(node.LastChild.Literal) > 0 {
				f = append(f, Fragment{
					Token: string(node.LastChild.Literal),
					Style: LinkText,
				})
			}
		}
		if len(node.LinkData.Destination) > 0 {
			f = append(f, Fragment{
				Token: string(node.LinkData.Destination),
				Pre:   " (",
				Post:  ")",
				Style: Link,
			})
		}

		return Element{
			Pre:       "",
			Post:      "",
			Fragments: f,
		}

	case bf.Image:
		f := []Fragment{}
		if len(node.LastChild.Literal) > 0 {
			f = append(f, Fragment{
				Token: string(node.LastChild.Literal),
				Style: Image,
			})
		}
		if len(node.LinkData.Destination) > 0 {
			f = append(f, Fragment{
				Token: string(node.LinkData.Destination),
				Pre:   " [Image: ",
				Post:  "]",
				Style: Link,
			})
		}

		return Element{
			Pre:       "",
			Post:      "",
			Fragments: f,
		}

	case bf.Text:
		return Element{
			Pre:  "",
			Post: "",
			Fragments: []Fragment{{
				Token: string(node.Literal),
				Style: Text,
			}},
		}
	case bf.HTMLBlock:
		return Element{
			Pre:  "\n",
			Post: "\n",
			Fragments: []Fragment{{
				Token: string(node.Literal),
				Style: HTMLBlock,
			}},
		}
	case bf.CodeBlock:
		return Element{
			Pre:  "\n",
			Post: "\n",
			Fragments: []Fragment{{
				Token: string(node.Literal),
				Style: CodeBlock,
			}},
		}
	case bf.Softbreak:
		return Element{
			Pre:  "",
			Post: "\n",
			Fragments: []Fragment{{
				Token: string(node.Literal),
				Style: Softbreak,
			}},
		}
	case bf.Hardbreak:
		return Element{
			Pre:  "\n",
			Post: "\n",
			Fragments: []Fragment{{
				Token: string(node.Literal),
				Style: Hardbreak,
			}},
		}
	case bf.Code:
		return Element{
			Pre:  "",
			Post: "",
			Fragments: []Fragment{{
				Token: string(node.Literal),
				Style: Code,
			}},
		}
	case bf.HTMLSpan:
		return Element{
			Pre:  "",
			Post: "",
			Fragments: []Fragment{{
				Token: string(node.Literal),
				Style: HTMLSpan,
			}},
		}
	case bf.Table:
		return Element{
			Pre:  "\n",
			Post: "\n",
			Fragments: []Fragment{{
				Token: string(node.Literal),
				Style: Table,
			}},
		}
	case bf.TableCell:
		return Element{
			Pre:  "",
			Post: "",
			Fragments: []Fragment{{
				Token: string(node.Literal),
				Style: TableCell,
			}},
		}
	case bf.TableHead:
		return Element{
			Pre:  "",
			Post: "",
			Fragments: []Fragment{{
				Token: string(node.Literal),
				Style: TableHead,
			}},
		}
	case bf.TableBody:
		return Element{
			Pre:  "",
			Post: "",
			Fragments: []Fragment{{
				Token: string(node.Literal),
				Style: TableBody,
			}},
		}
	case bf.TableRow:
		return Element{
			Pre:  "\n",
			Post: "\n",
			Fragments: []Fragment{{
				Token: string(node.Literal),
				Style: TableRow,
			}},
		}

	default:
		return Element{
			Pre:  "",
			Post: "",
			Fragments: []Fragment{{
				Token: string(node.Literal),
			}},
		}
	}
}

func (tr *TermRenderer) Render(in string) string {
	return string(tr.RenderBytes([]byte(in)))
}

func (tr *TermRenderer) RenderBytes(in []byte) []byte {
	return bf.Run(in, bf.WithRenderer(tr))
}

func (tr *TermRenderer) renderFragment(w io.Writer, f Fragment) {
	rules := tr.style[f.Style]
	if rules == nil {
		fmt.Fprintf(w, "%s", f.Token)
		return
	}

	out := aurora.Reset(f.Token)
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
}

func (tr *TermRenderer) RenderNode(w io.Writer, node *bf.Node, entering bool) bf.WalkStatus {
	// fmt.Fprintf(w, "%s %t", node.Type, entering)
	e := NewElement(node)
	if entering && e.Pre != "" {
		fmt.Fprintf(w, "%s", e.Pre)
	}
	if !entering && e.Post != "" {
		fmt.Fprintf(w, "%s", e.Post)
	}
	if isChild(node) {
		return bf.GoToNext
	}
	if !entering {
		return bf.GoToNext
	}

	for _, f := range e.Fragments {
		if f.Token == "" {
			continue
		}

		if f.Pre != "" {
			fmt.Fprintf(w, "%s", f.Pre)
		}
		tr.renderFragment(w, f)
		if f.Post != "" {
			fmt.Fprintf(w, "%s", f.Post)
		}
	}

	return bf.GoToNext
}

func (tr *TermRenderer) RenderHeader(w io.Writer, ast *bf.Node) {
}

func (tr *TermRenderer) RenderFooter(w io.Writer, ast *bf.Node) {
}

func isChild(node *bf.Node) bool {
	if node.Parent == nil {
		return false
	}
	switch node.Parent.Type {
	case bf.Heading, bf.Link, bf.Image, bf.Emph, bf.Strong:
		return true
	default:
		return false
	}
}
