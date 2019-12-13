package ansi

import (
	"io"
	"log"
	"net/url"
	"strings"

	"github.com/yuin/goldmark/ast"
	astext "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type Options struct {
	BaseURL  string
	WordWrap int
	Styles   StyleConfig
}

type ANSIRenderer struct {
	context RenderContext
}

// NewANSIRenderer returns a new ANSIRenderer with style and options set.
func NewRenderer(options Options) *ANSIRenderer {
	return &ANSIRenderer{
		context: NewRenderContext(options),
	}
}

// RegisterFuncs implements NodeRenderer.RegisterFuncs.
func (r *ANSIRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
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

func (tr *ANSIRenderer) renderNode(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
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
	case ast.KindLink, ast.KindImage, ast.KindEmphasis, astext.KindStrikethrough, ast.KindBlockquote, astext.KindTableCell:
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
