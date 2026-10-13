package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSourceFromArgURLAcceptsMarkdown(t *testing.T) {
	var accept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept = r.Header.Get("Accept")
		_, _ = io.WriteString(w, "# hello\n")
	}))
	defer srv.Close()

	src, err := sourceFromArg(srv.URL + "/page")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer src.reader.Close() //nolint: errcheck

	body, err := io.ReadAll(src.reader)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if string(body) != "# hello\n" {
		t.Errorf("unexpected body %q", body)
	}
	if !strings.HasPrefix(accept, "text/markdown") {
		t.Errorf("expected an Accept header preferring text/markdown, got %q", accept)
	}
}
