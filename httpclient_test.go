package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestHTTPClientHasTimeout guards against regressing back to a client with no
// timeout, which would let an unresponsive host hang glow indefinitely.
func TestHTTPClientHasTimeout(t *testing.T) {
	if httpClient.Timeout <= 0 {
		t.Fatalf("httpClient must have a positive timeout, got %v", httpClient.Timeout)
	}
}

// TestSourceFromArgHTTPTimeout verifies that fetching a remote source honors
// the client timeout instead of blocking forever on an unresponsive host.
func TestSourceFromArgHTTPTimeout(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		<-block // never respond until the test tears the server down
	}))
	defer srv.Close()
	defer close(block) // runs first (LIFO): unblock handler before srv.Close()

	orig := httpClient.Timeout
	httpClient.Timeout = 100 * time.Millisecond
	defer func() { httpClient.Timeout = orig }()

	start := time.Now()
	if _, err := sourceFromArg(srv.URL); err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("request did not honor timeout, took %s", elapsed)
	}
}
