package main

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
)

// isGitHubURL tests a string to determine if it is a well-structured GitHub URL.
func isGitHubURL(s string) (string, bool) {
	if strings.HasPrefix(s, "github.com/") {
		s = "https://" + s
	}

	u, err := url.ParseRequestURI(s)
	if err != nil {
		return "", false
	}

	return u.String(), strings.ToLower(u.Host) == "github.com"
}

// findGitHubREADME tries to find the correct README filename in a repository.
func findGitHubREADME(s string) (*source, error) {
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return nil, err
	}
	u.Host = "raw.githubusercontent.com"

	for _, r := range readmeNames {
		v := u
		v.Path += "/master/" + r

		resp, err := http.Get(v.String())
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusOK {
			return &source{resp.Body, v.String()}, nil
		}
	}

	return nil, errors.New("can't find README in GitHub repository")
}
