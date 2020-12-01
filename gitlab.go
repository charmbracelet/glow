package main

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
)

// isGitLabURL tests a string to determine if it is a well-structured GitLab URL.
func isGitLabURL(s string) (string, bool) {
	if strings.HasPrefix(s, "gitlab.com/") {
		s = "https://" + s
	}

	u, err := url.ParseRequestURI(s)
	if err != nil {
		return "", false
	}

	return u.String(), strings.ToLower(u.Host) == "gitlab.com"
}

// findGitLabREADME tries to find the correct README filename in a repository.
func findGitLabREADME(s string) (*source, error) {
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return nil, err
	}

	for _, r := range readmeNames {
		v := u
		v.Path += "/raw/master/" + r

		resp, err := http.Get(v.String())
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusOK {
			return &source{resp.Body, v.String()}, nil
		}
	}

	return nil, errors.New("can't find README in GitLab repository")
}
