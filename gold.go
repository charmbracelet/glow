package gold

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"

	"github.com/microcosm-cc/bluemonday"
	"github.com/olekukonko/tablewriter"
	bf "gopkg.in/russross/blackfriday.v2"
)

var (
	stripper = bluemonday.StrictPolicy()
)

type TableData struct {
	table       *tablewriter.Table
	tableHeader []string
	tableCell   []string
}

type TermRenderer struct {
	BaseURL   string
	style     map[StyleType]*ElementStyle
	tableData TableData
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

func (tr *TermRenderer) Render(in string) string {
	return string(tr.RenderBytes([]byte(in)))
}

func (tr *TermRenderer) RenderBytes(in []byte) []byte {
	return bf.Run(in, bf.WithRenderer(tr))
}

func (tr *TermRenderer) RenderNode(w io.Writer, node *bf.Node, entering bool) bf.WalkStatus {
	// fmt.Fprintf(w, "%s %t", node.Type, entering)
	e := tr.NewElement(node)
	if entering && e.Entering != "" {
		fmt.Fprintf(w, "%s", e.Entering)
	}
	if !entering && e.Exiting != "" {
		fmt.Fprintf(w, "%s", e.Exiting)
	}

	if !entering {
		if e.Finisher != nil {
			err := e.Finisher.Finish(w, node, tr)
			if err != nil {
				fmt.Println(err)
				return bf.Terminate
			}
		}
		return bf.GoToNext
	}

	if isChild(node) {
		return bf.GoToNext
	}

	if e.Renderer != nil {
		err := e.Renderer.Render(w, node, tr)
		if err != nil {
			fmt.Println(err)
			return bf.Terminate
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
	case bf.Heading, bf.Link, bf.Image, bf.TableCell, bf.Emph, bf.Strong:
		return true
	default:
		return false
	}
}

func resolveRelativeURL(baseURL string, rel string) string {
	u, err := url.Parse(rel)
	if err != nil {
		log.Fatal(err)
	}
	if u.IsAbs() {
		return rel
	}
	if u.Path[0] == '/' {
		u.Path = u.Path[1:]
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return rel
	}
	return base.ResolveReference(u).String()
}
