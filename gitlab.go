package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// findGitLabREADME tries to find the correct README filename in a repository using GitLab API.
func findGitLabREADME(u *url.URL) (*source, error) {
	owner, repo, ok := strings.Cut(strings.TrimPrefix(u.Path, "/"), "/")
	if !ok {
		return nil, fmt.Errorf("invalid url: %s", u.String())
	}

	projectPath := url.QueryEscape(owner + "/" + repo)

	type readme struct {
		ReadmeURL string `json:"readme_url"`
	}

	apiURL := fmt.Sprintf("https://%s/api/v4/projects/%s", u.Hostname(), projectPath)

	// nolint:bodyclose
	// it is closed on the caller
	res, err := http.Get(apiURL) // nolint: gosec
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var result readme
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	readmeRawURL := strings.ReplaceAll(result.ReadmeURL, "blob", "raw")

	if res.StatusCode == http.StatusOK {
		// nolint:bodyclose
		// it is closed on the caller
		resp, err := http.Get(readmeRawURL) // nolint: gosec
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusOK {
			return &source{resp.Body, readmeRawURL}, nil
		}
	}

	return nil, errors.New("can't find README in GitLab repository")
}
