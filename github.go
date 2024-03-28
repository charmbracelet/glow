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

// findGitHubREADME tries to find the correct README filename in a repository using GitHub API.
func findGitHubREADME(u *url.URL) (*source, error) {
	owner, repo, ok := strings.Cut(strings.TrimPrefix(u.Path, "/"), "/")
	if !ok {
		return nil, fmt.Errorf("invalid url: %s", u.String())
	}

	type readme struct {
		DownloadURL string `json:"download_url"`
	}

	apiURL := fmt.Sprintf("https://api.%s/repos/%s/%s/readme", u.Hostname(), owner, repo)

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

	if res.StatusCode == http.StatusOK {
		// nolint:bodyclose
		// it is closed on the caller
		resp, err := http.Get(result.DownloadURL)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusOK {
			return &source{resp.Body, result.DownloadURL}, nil
		}
	}

	return nil, errors.New("can't find README in GitHub repository")
}
