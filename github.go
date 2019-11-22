package main

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
)

// isGitHubURL tests a string to determine if it is a well-structured GitHub URL
func isGitHubURL(s string) bool {
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return false
	}

	return strings.ToLower(u.Host) == "github.com"
}

// findGitHubREADME tries to find the correct README filename in a repository
func findGitHubREADME(s string) (*http.Response, error) {
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return nil, err
	}

	u.Host = "raw.githubusercontent.com"
	readmeNames := []string{"README.md", "README"}

	for _, r := range readmeNames {
		v := u
		v.Path += "/master/" + r

		resp, err := http.Get(v.String())
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusOK {
			return resp, nil
		}
	}

	return nil, errors.New("can't find README in GitHub repository")
}
