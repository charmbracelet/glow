package main

import (
	"encoding/json"
	"errors"
	"io"
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

// findGitHubREADME tries to find the correct README filename in a repository using GitHub API.
func findGitHubREADME(s string) (*source, error) {
	sSplit := strings.Split(s, "/")
	owner, repo := sSplit[3], sSplit[4]

	type readme struct {
		DownloadURL string `json:"download_url"`
	}

	apiURL := "https://api.github.com/repos/" + owner + "/" + repo + "/readme"

	// nolint:bodyclose
	// it is closed on the caller
	res, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var result readme
	jsonErr := json.Unmarshal(body, &result)
	if jsonErr != nil {
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
