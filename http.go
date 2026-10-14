package main

import (
	"fmt"
	"net/http"
)

func httpClient() *http.Client {
	return &http.Client{
		Transport: userAgentRoundTripper{
			roundTripper: http.DefaultTransport,
		},
	}
}

type userAgentRoundTripper struct {
	roundTripper http.RoundTripper
}

func (u userAgentRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		version := Version
		if version == "" {
			version = "unknown"
		}
		req.Header.Set("User-Agent", fmt.Sprintf("glow/%s", version))
	}
	rt := u.roundTripper
	if rt == nil {
		rt = http.DefaultTransport
	}
	return rt.RoundTrip(req)
}
