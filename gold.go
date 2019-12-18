package gold

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/rakyll/statik/fs"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"

	"github.com/charmbracelet/gold/ansi"
	_ "github.com/charmbracelet/gold/statik"
)

type TermRenderer struct {
	md        goldmark.Markdown
	buf       bytes.Buffer
	renderBuf bytes.Buffer
}

// Render initializes a new TermRenderer and renders a markdown with a specific
// style.
func Render(in string, stylePath string) (string, error) {
	b, err := RenderBytes([]byte(in), stylePath)
	return string(b), err
}

// RenderBytes initializes a new TermRenderer and renders a markdown with a
// specific style.
func RenderBytes(in []byte, stylePath string) ([]byte, error) {
	r, err := NewTermRenderer(stylePath, ansi.Options{
		WordWrap: 80,
	})
	if err != nil {
		return nil, err
	}
	return r.RenderBytes(in)
}

// NewTermRenderer returns a new TermRenderer with style and options set.
func NewTermRenderer(stylePath string, options ansi.Options) (*TermRenderer, error) {
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
func NewTermRendererFromBytes(b []byte, options ansi.Options) (*TermRenderer, error) {
	err := json.Unmarshal(b, &options.Styles)
	if err != nil {
		// FIXME: wrap error once we depend on Go 1.13
		return nil, fmt.Errorf("parsing style: %v", err)
	}

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.DefinitionList,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	ar := ansi.NewRenderer(options)
	md.SetRenderer(
		renderer.NewRenderer(
			renderer.WithNodeRenderers(util.Prioritized(ar, 1000))))

	return &TermRenderer{
		md: md,
	}, nil
}

func (tr *TermRenderer) Read(b []byte) (int, error) {
	return tr.renderBuf.Read(b)
}

func (tr *TermRenderer) Write(b []byte) (int, error) {
	return tr.buf.Write(b)
}

func (tr *TermRenderer) Close() error {
	err := tr.md.Convert(tr.buf.Bytes(), &tr.renderBuf)
	if err != nil {
		return err
	}

	tr.buf.Reset()
	return nil
}

// Render returns the markdown rendered into a string.
func (tr *TermRenderer) Render(in string) (string, error) {
	b, err := tr.RenderBytes([]byte(in))
	return string(b), err
}

// RenderBytes returns the markdown rendered into a byte slice.
func (tr *TermRenderer) RenderBytes(in []byte) ([]byte, error) {
	var buf bytes.Buffer
	err := tr.md.Convert(in, &buf)
	return buf.Bytes(), err
}

func loadStyle(f string) ([]byte, error) {
	var r io.ReadCloser

	r, err := os.Open(f)
	if err != nil {
		statikFS, err := fs.New()
		if err != nil {
			return nil, err
		}

		r, err = statikFS.Open("/" + f + ".json")
		if err != nil {
			// FIXME: wrap error once we depend on Go 1.13
			return nil, fmt.Errorf("loading style %s: %v", f, err)
		}
	}

	defer r.Close()
	return ioutil.ReadAll(r)
}
