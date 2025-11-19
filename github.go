package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// findGitHubREADME tries to find the correct README filename in a repository using GitHub API.
func findGitHubREADME(ctx context.Context, u *url.URL) (*source, error) {
	owner, repo, ok := strings.Cut(strings.TrimPrefix(u.Path, "/"), "/")
	if !ok {
		return nil, fmt.Errorf("invalid url: %s", u.String())
	}

	type readme struct {
		DownloadURL string `json:"download_url"`
	}

	apiURL := fmt.Sprintf("https://api.%s/repos/%s/%s/readme", u.Hostname(), owner, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %w", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to get url: %w", err)
	}
	defer res.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read http response body: %w", err)
	}

	var result readme
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unable to parse json: %w", err)
	}

	if res.StatusCode == http.StatusOK {
		// consumer of the source is responsible for closing the ReadCloser.
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, result.DownloadURL, nil)
		if err != nil {
			return nil, fmt.Errorf("unable to create request: %w", err)
		}
		resp, err := http.DefaultClient.Do(req) //nolint:bodyclose
		if err != nil {
			return nil, fmt.Errorf("unable to get url: %w", err)
		}

		if resp.StatusCode == http.StatusOK {
			return &source{resp.Body, result.DownloadURL}, nil
		}
	}

	return nil, errors.New("can't find README in GitHub repository")
}
