package gold

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"

	"github.com/microcosm-cc/bluemonday"
	bf "gopkg.in/russross/blackfriday.v2"
)

var (
	stripper = bluemonday.StrictPolicy()
)

type Options struct {
	BaseURL  string
	WordWrap int
}

type TermRenderer struct {
	context RenderContext
}

func Render(in string, stylePath string) ([]byte, error) {
	return RenderBytes([]byte(in), stylePath)
}

func RenderBytes(in []byte, stylePath string) ([]byte, error) {
	r, err := NewTermRenderer(stylePath, Options{
		WordWrap: 80,
	})
	if err != nil {
		return nil, err
	}
	return r.RenderBytes(in), nil
}

func NewPlainTermRenderer(options Options) *TermRenderer {
	return &TermRenderer{
		context: RenderContext{
			style:      make(map[StyleType]ElementStyle),
			blockStack: &BlockStack{},
			table:      &TableElement{},
			options:    options,
		},
	}
}

func NewTermRenderer(stylePath string, options Options) (*TermRenderer, error) {
	if stylePath == "" {
		return NewTermRendererFromBytes([]byte("{}"), options)
	}

	b, err := loadStyle(stylePath)
	if err != nil {
		return nil, err
	}
	return NewTermRendererFromBytes(b, options)
}

func NewTermRendererFromBytes(b []byte, options Options) (*TermRenderer, error) {
	e := make(map[string]ElementStyle)
	err := json.Unmarshal(b, &e)
	if err != nil {
		return nil, err
	}

	tr := NewPlainTermRenderer(options)
	for k, v := range e {
		t, err := keyToType(k)
		if err != nil {
			fmt.Println(err)
			continue
		}
		tr.context.style[t] = v
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
	// _, _ = w.Write([]byte(node.Type.String()))
	writeTo := w
	bs := tr.context.blockStack

	if isChild(node) {
		return bf.GoToNext
	}

	e := tr.NewElement(node)
	if entering {
		if bs.Len() > 0 {
			writeTo = io.Writer(bs.Current().Block)
		}
		_, _ = writeTo.Write([]byte(e.Entering))

		if e.Renderer != nil {
			err := e.Renderer.Render(writeTo, tr.context)
			if err != nil {
				fmt.Println(err)
				return bf.Terminate
			}
		}
	} else {
		if bs.Len() > 0 {
			writeTo = io.Writer(bs.Parent().Block)
		}

		// if we're finished rendering the entire document,
		// flush to the real writer
		if node.Type == bf.Document {
			writeTo = w
		}

		if e.Finisher != nil {
			err := e.Finisher.Finish(writeTo, tr.context)
			if err != nil {
				fmt.Println(err)
				return bf.Terminate
			}
		}

		_, _ = writeTo.Write([]byte(e.Exiting))
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
