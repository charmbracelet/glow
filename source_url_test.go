package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestExecuteCLISourceURL(t *testing.T) {
	oldStyle, oldWidth, oldPager, oldTUI := style, width, pager, tui
	t.Cleanup(func() {
		style, width, pager, tui = oldStyle, oldWidth, oldPager, oldTUI
	})
	style, width, pager, tui = "light", 60, false, false

	render := func(name, content string) string {
		t.Helper()
		cmd := &cobra.Command{}
		cmd.Flags().Bool("pager", false, "")
		cmd.Flags().Bool("tui", false, "")
		var output bytes.Buffer
		src := &source{reader: io.NopCloser(strings.NewReader(content)), URL: name}
		if err := executeCLI(cmd, src, &output); err != nil {
			t.Fatal(err)
		}
		return output.String()
	}

	const markdown = "# Release notes\n\nThis is **formatted Markdown**.\n"
	const code = "package main\n\nfunc main() {}\n"
	tests := []struct {
		name, sourceURL, filename, content string
	}{
		{"markdown URL", "https://example.com/README.md", "README.md", markdown},
		{"query", "https://example.com/README.md?version=2", "README.md", markdown},
		{"fragment", "https://example.com/README.md#release", "README.md", markdown},
		{"query and fragment", "https://example.com/README.md?version=2#release", "README.md", markdown},
		{"extension in query", "https://example.com/README.md?download=source.go", "README.md", markdown},
		{"encoded path", "https://example.com/README%2Emd?version=2", "README.md", markdown},
		{"extensionless URL", "https://example.com/readme?format=.go", "readme", markdown},
		{"code URL", "https://example.com/main.go", "main.go", code},
		{"code query", "https://example.com/main.go?version=2", "main.go", code},
		{"code fragment", "https://example.com/main.go#L1", "main.go", code},
		{"local question mark", "/notes/README.md?draft", "README.md?draft", markdown},
		{"local hash", "/notes/README#copy.md", "README#copy.md", markdown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := render(tt.filename, tt.content)
			got := render(tt.sourceURL, tt.content)
			if got != want {
				t.Errorf("source %q renders differently from %q", tt.sourceURL, tt.filename)
			}
		})
	}
}

func TestSourceFromArgURLFragment(t *testing.T) {
	requests := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.URL.RequestURI()
		_, _ = io.WriteString(w, "# Release notes\n")
	}))
	defer server.Close()

	for _, tt := range []struct {
		name, path, wantRequest string
	}{
		{"fragment only", "/README.md#release", "/README.md"},
		{"fragment", "/README.md?version=2#release", "/README.md?version=2"},
		{"encoded hash in filename", "/README%23copy.md?version=2", "/README%23copy.md?version=2"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			src, err := sourceFromArg(server.URL + tt.path)
			if err != nil {
				t.Fatal(err)
			}
			defer src.reader.Close()
			if got := <-requests; got != tt.wantRequest {
				t.Errorf("request URI = %q, want %q", got, tt.wantRequest)
			}
		})
	}
}
