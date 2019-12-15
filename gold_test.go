package gold

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/charmbracelet/gold/ansi"
)

const (
	generate = false
	markdown = "testdata/readme.markdown.in"
	testFile = "testdata/readme.test"
)

func TestTermRenderer(t *testing.T) {
	r, err := NewTermRenderer("styles/dark.json", ansi.Options{
		WordWrap: 80,
	})
	if err != nil {
		t.Fatal(err)
	}

	in, err := ioutil.ReadFile(markdown)
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Write(in)
	if err != nil {
		t.Fatal(err)
	}
	err = r.Close()
	if err != nil {
		t.Fatal(err)
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	// generate
	if generate {
		err := ioutil.WriteFile(testFile, b, 0644)
		if err != nil {
			t.Fatal(err)
		}
		return
	}

	// verify
	td, err := ioutil.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(td, b) {
		t.Errorf("Rendered output doesn't match!\nExpected: `\n%s`\nGot: `\n%s`\n",
			string(td), b)
	}
}
