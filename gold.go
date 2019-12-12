package gold

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	astext "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type Options struct {
	BaseURL  string
	WordWrap int
}

type TermRenderer struct {
	context RenderContext
}

// Render initializes a new TermRenderer and renders a markdown with a specific
// style.
func Render(in string, stylePath string) ([]byte, error) {
	return RenderBytes([]byte(in), stylePath)
}

// RenderBytes initializes a new TermRenderer and renders a markdown with a
// specific style.
func RenderBytes(in []byte, stylePath string) ([]byte, error) {
	r, err := NewTermRenderer(stylePath, Options{
		WordWrap: 80,
	})
	if err != nil {
		return nil, err
	}
	return r.RenderBytes(in), nil
}

// NewTermRenderer returns a new TermRenderer with style and options set.
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

// NewTermRendererFromBytes returns a new TermRenderer with style and options
// set.
func NewTermRendererFromBytes(b []byte, options Options) (*TermRenderer, error) {
	e := make(map[string]ElementStyle)
	err := json.Unmarshal(b, &e)
	if err != nil {
		return nil, err
	}

	tr := &TermRenderer{
		context: NewRenderContext(options),
	}

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

// Render returns the markdown rendered into a string.
func (tr *TermRenderer) Render(in string) string {
	return string(tr.RenderBytes([]byte(in)))
}

// RenderBytes returns the markdown rendered into a byte slice.
func (tr *TermRenderer) RenderBytes(in []byte) []byte {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)
	md.SetRenderer(
		renderer.NewRenderer(
			renderer.WithNodeRenderers(util.Prioritized(tr, 1000))))

	var buf bytes.Buffer
	if err := md.Convert(in, &buf); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// RegisterFuncs implements NodeRenderer.RegisterFuncs .
func (r *TermRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	// blocks
	reg.Register(ast.KindDocument, r.renderNode)
	reg.Register(ast.KindHeading, r.renderNode)
	reg.Register(ast.KindBlockquote, r.renderNode)
	reg.Register(ast.KindCodeBlock, r.renderNode)
	reg.Register(ast.KindFencedCodeBlock, r.renderNode)
	reg.Register(ast.KindHTMLBlock, r.renderNode)
	reg.Register(ast.KindList, r.renderNode)
	reg.Register(ast.KindListItem, r.renderNode)
	reg.Register(ast.KindParagraph, r.renderNode)
	reg.Register(ast.KindTextBlock, r.renderNode)
	reg.Register(ast.KindThematicBreak, r.renderNode)

	// inlines
	reg.Register(ast.KindAutoLink, r.renderNode)
	reg.Register(ast.KindCodeSpan, r.renderNode)
	reg.Register(ast.KindEmphasis, r.renderNode)
	reg.Register(ast.KindImage, r.renderNode)
	reg.Register(ast.KindLink, r.renderNode)
	reg.Register(ast.KindRawHTML, r.renderNode)
	reg.Register(ast.KindText, r.renderNode)
	reg.Register(ast.KindString, r.renderNode)

	// tables
	reg.Register(astext.KindTable, r.renderNode)
	reg.Register(astext.KindTableHeader, r.renderNode)
	reg.Register(astext.KindTableRow, r.renderNode)
	reg.Register(astext.KindTableCell, r.renderNode)

	// definitions
	reg.Register(astext.KindDefinitionList, r.renderNode)
	reg.Register(astext.KindDefinitionTerm, r.renderNode)
	reg.Register(astext.KindDefinitionDescription, r.renderNode)

	// footnotes
	reg.Register(astext.KindFootnote, r.renderNode)
	reg.Register(astext.KindFootnoteList, r.renderNode)
	reg.Register(astext.KindFootnoteLink, r.renderNode)
	reg.Register(astext.KindFootnoteBackLink, r.renderNode)

	// checkboxes
	reg.Register(astext.KindTaskCheckBox, r.renderNode)

	// strikethrough
	reg.Register(astext.KindStrikethrough, r.renderNode)
}

func (tr *TermRenderer) renderNode(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	// _, _ = w.Write([]byte(node.Type.String()))
	writeTo := io.Writer(w)
	bs := tr.context.blockStack

	// children get rendered by their parent
	if isChild(node) {
		return ast.WalkContinue, nil
	}

	e := tr.NewElement(node, source)
	if entering {
		// everything below the Document element gets rendered into a block buffer
		if bs.Len() > 0 {
			writeTo = io.Writer(bs.Current().Block)
		}

		_, _ = writeTo.Write([]byte(e.Entering))
		if e.Renderer != nil {
			err := e.Renderer.Render(writeTo, tr.context)
			if err != nil {
				return ast.WalkStop, err
			}
		}
	} else {
		// everything below the Document element gets rendered into a block buffer
		if bs.Len() > 0 {
			writeTo = io.Writer(bs.Parent().Block)
		}

		// if we're finished rendering the entire document,
		// flush to the real writer
		if node.Type() == ast.TypeDocument {
			writeTo = w
		}

		if e.Finisher != nil {
			err := e.Finisher.Finish(writeTo, tr.context)
			if err != nil {
				return ast.WalkStop, err
			}
		}
		_, _ = bs.Current().Block.Write([]byte(e.Exiting))
	}

	return ast.WalkContinue, nil
}

func isChild(node ast.Node) bool {
	if node.Parent() == nil {
		return false
	}

	// These types are already rendered by their parent
	switch node.Parent().Kind() {
	case ast.KindLink, ast.KindImage, ast.KindEmphasis, ast.KindBlockquote, astext.KindTableCell:
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
	u.Path = strings.TrimPrefix(u.Path, "/")

	base, err := url.Parse(baseURL)
	if err != nil {
		return rel
	}
	return base.ResolveReference(u).String()
}
