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

// findGitLabREADME tries to find the correct README filename in a repository using GitLab API.
func findGitLabREADME(ctx context.Context, u *url.URL) (*source, error) {
	owner, repo, ok := strings.Cut(strings.TrimPrefix(u.Path, "/"), "/")
	if !ok {
		return nil, fmt.Errorf("invalid url: %s", u.String())
	}

	projectPath := url.QueryEscape(owner + "/" + repo)

	type readme struct {
		ReadmeURL string `json:"readme_url"`
	}

	apiURL := fmt.Sprintf("https://%s/api/v4/projects/%s", u.Hostname(), projectPath)

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

	readmeRawURL := strings.ReplaceAll(result.ReadmeURL, "blob", "raw")

	if res.StatusCode == http.StatusOK {
		// consumer of the source is responsible for closing the ReadCloser.
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, readmeRawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("unable to create request: %w", err)
		}
		resp, err := http.DefaultClient.Do(req) //nolint:bodyclose
		if err != nil {
			return nil, fmt.Errorf("unable to get url: %w", err)
		}

		if resp.StatusCode == http.StatusOK {
			return &source{resp.Body, readmeRawURL}, nil
		}
	}

	return nil, errors.New("can't find README in GitLab repository")
}
