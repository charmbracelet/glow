package gold

import (
	"bytes"
	"encoding/json"
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
	md goldmark.Markdown
}

// Render initializes a new TermRenderer and renders a markdown with a specific
// style.
func Render(in string, stylePath string) ([]byte, error) {
	return RenderBytes([]byte(in), stylePath)
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
		return nil, err
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
	var err error

	r, err = os.Open(f)
	if err != nil {
		statikFS, err := fs.New()
		if err != nil {
			return nil, err
		}

		r, err = statikFS.Open("/" + f + ".json")
		if err != nil {
			return nil, err
		}
	}

	defer r.Close()
	return ioutil.ReadAll(r)
}
