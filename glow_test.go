package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGlowFlags(t *testing.T) {
	tt := []struct {
		args  []string
		check func() bool
	}{
		{
			args: []string{"-p"},
			check: func() bool {
				return pager
			},
		},
		{
			args: []string{"-s", "light"},
			check: func() bool {
				return style == "light"
			},
		},
		{
			args: []string{"-w", "40"},
			check: func() bool {
				return width == 40
			},
		},
	}

	for _, v := range tt {
		err := rootCmd.ParseFlags(v.args)
		if err != nil {
			t.Fatal(err)
		}
		if !v.check() {
			t.Errorf("Parsing flag failed: %s", v.args)
		}
	}
}

func TestSourceDetectsExtensionlessJSONContentType(t *testing.T) {
	src := source{
		URL:         "https://api.example.com/endpoint",
		contentType: "application/vnd.api+json; charset=utf-8",
	}

	if !src.isCode() {
		t.Fatal("expected extensionless JSON response to render as code")
	}
	if got := src.codeBlockLanguage(); got != ".json" {
		t.Fatalf("expected .json code block language, got %q", got)
	}
}

func TestSourceMarkdownExtensionTakesPrecedenceOverContentType(t *testing.T) {
	src := source{
		URL:         "https://example.com/README.md",
		contentType: "application/json",
	}

	if src.isCode() {
		t.Fatal("expected markdown extension to render as markdown")
	}
}

func TestSourceFromArgRecordsHTTPContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprint(w, `{"ok":true}`)
	}))
	t.Cleanup(server.Close)

	src, err := sourceFromArg(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := src.reader.Close(); err != nil {
			t.Fatal(err)
		}
	})

	if got := src.contentType; got != "application/json; charset=utf-8" {
		t.Fatalf("expected content type to be recorded, got %q", got)
	}
	if !src.isCode() {
		t.Fatal("expected extensionless JSON HTTP source to render as code")
	}
}
