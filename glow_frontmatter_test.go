package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestExecuteCLIPreservesSourceDocuments(t *testing.T) {
	oldStyle, oldWidth, oldPager, oldTUI := style, width, pager, tui
	t.Cleanup(func() { style, width, pager, tui = oldStyle, oldWidth, oldPager, oldTUI })
	style, width, pager, tui = "notty", 80, false, false

	const document = "---\nname: first-config\n---\nname: second-config\n"
	for _, tc := range []struct {
		filename  string
		wantFirst bool
	}{
		{filename: "manifests.yaml", wantFirst: true},
		{filename: "manifests.yml", wantFirst: true},
		{filename: "manifests.YAML", wantFirst: true},
		{filename: "notes.txt", wantFirst: true},
		{filename: "article.md", wantFirst: false},
		{filename: "article.MD", wantFirst: false},
		{filename: "README", wantFirst: false},
		{filename: "", wantFirst: false},
	} {
		t.Run(tc.filename, func(t *testing.T) {
			src := &source{reader: io.NopCloser(strings.NewReader(document)), URL: tc.filename}
			var out bytes.Buffer
			if err := executeCLI(&cobra.Command{}, src, &out); err != nil {
				t.Fatal(err)
			}
			if got := strings.Contains(out.String(), "first-config"); got != tc.wantFirst {
				t.Errorf("first document visible = %t, want %t; output: %q", got, tc.wantFirst, out.String())
			}
			if !strings.Contains(out.String(), "second-config") {
				t.Errorf("second document missing from output: %q", out.String())
			}
		})
	}
}
