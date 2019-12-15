package gold

import (
	"io/ioutil"
	"testing"

	"github.com/charmbracelet/gold/ansi"
)

func TestTermRenderer(t *testing.T) {
	r, err := NewTermRenderer("notty", ansi.Options{
		WordWrap: 80,
	})
	if err != nil {
		t.Fatal(err)
	}

	expLen := 166
	_, err = r.Write([]byte("# Test"))
	if err != nil {
		t.Error(err)
	}
	err = r.Close()
	if err != nil {
		t.Error(err)
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		t.Error(err)
	}
	if len(b) != expLen {
		t.Errorf("Expected rendered content with a length of %d, got %d!", expLen, len(b))
	}
}
