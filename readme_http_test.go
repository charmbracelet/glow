package main

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type readmeRoundTripFunc func(*http.Request) (*http.Response, error)

func (f readmeRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type trackedReadCloser struct {
	io.Reader
	closed bool
}

func (b *trackedReadCloser) Close() error {
	b.closed = true
	return nil
}

func TestGitHubREADMEClosesResponsesOnFailure(t *testing.T) {
	apiBody := &trackedReadCloser{Reader: strings.NewReader(`{"download_url":"https://raw.githubusercontent.com/example/repo/README.md"}`)}
	rawBody := &trackedReadCloser{Reader: strings.NewReader("not found")}
	oldClient := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: readmeRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "api.github.com" {
			return &http.Response{StatusCode: http.StatusOK, Body: apiBody}, nil
		}
		return &http.Response{StatusCode: http.StatusNotFound, Body: rawBody}, nil
	})}
	t.Cleanup(func() { http.DefaultClient = oldClient })

	u, _ := url.Parse("https://github.com/example/repo")
	if _, err := findGitHubREADME(u); err == nil {
		t.Fatal("expected a missing README error")
	}
	if !apiBody.closed || !rawBody.closed {
		t.Fatalf("response bodies were not closed: API=%v raw=%v", apiBody.closed, rawBody.closed)
	}
}

func TestGitLabREADMEPreservesBlobInNames(t *testing.T) {
	const rawURL = "https://gitlab.com/blob-team/repo/-/raw/main/blob-readme.md"
	apiBody := &trackedReadCloser{Reader: strings.NewReader(`{"readme_url":"https://gitlab.com/blob-team/repo/-/blob/main/blob-readme.md"}`)}
	rawBody := &trackedReadCloser{Reader: strings.NewReader("# README")}
	oldClient := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: readmeRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "/api/v4/projects/") {
			return &http.Response{StatusCode: http.StatusOK, Body: apiBody}, nil
		}
		if r.URL.String() != rawURL {
			t.Errorf("unexpected raw URL: %s", r.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: rawBody}, nil
	})}
	t.Cleanup(func() { http.DefaultClient = oldClient })

	u, _ := url.Parse("https://gitlab.com/blob-team/repo")
	source, err := findGitLabREADME(u)
	if err != nil {
		t.Fatal(err)
	}
	if source == nil || source.URL != rawURL {
		t.Fatalf("unexpected README source: %+v", source)
	}
	if !apiBody.closed || rawBody.closed {
		t.Fatalf("unexpected body ownership: API=%v raw=%v", apiBody.closed, rawBody.closed)
	}
	if err := source.reader.Close(); err != nil {
		t.Fatal(err)
	}
}
