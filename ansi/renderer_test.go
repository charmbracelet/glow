package ansi

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

const (
	examplesDir = "../styles/examples/"
	generate    = false
)

func TestRenderer(t *testing.T) {
	files, err := filepath.Glob(examplesDir + "*.md")
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		bn := strings.TrimSuffix(filepath.Base(f), ".md")
		sn := filepath.Join(examplesDir, bn+".style")
		tn := filepath.Join(examplesDir, bn+".test")

		in, err := ioutil.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		b, err := ioutil.ReadFile(sn)
		if err != nil {
			t.Fatal(err)
		}

		options := Options{
			WordWrap: 80,
		}
		err = json.Unmarshal(b, &options.Styles)
		if err != nil {
			t.Fatal(err)
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

		ar := NewRenderer(options)
		md.SetRenderer(
			renderer.NewRenderer(
				renderer.WithNodeRenderers(util.Prioritized(ar, 1000))))

		var buf bytes.Buffer
		err = md.Convert(in, &buf)
		if err != nil {
			t.Error(err)
		}

		// generate
		if generate {
			err = ioutil.WriteFile(tn, buf.Bytes(), 0644)
			if err != nil {
				t.Fatal(err)
			}
			continue
		}

		// verify
		td, err := ioutil.ReadFile(tn)
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(td, buf.Bytes()) {
			t.Errorf("Rendered output for %s doesn't match!\nExpected: `\n%s`\nGot: `\n%s`\n",
				bn, string(td), buf.String())
		}
	}
}
