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

// findGitHubREADME finds the correct README filename in a repository using GitHub API.
func findGitHubREADME(s string) (*source, error) {
	sSplit := strings.Split(s, "/")
	owner, repo := sSplit[3], sSplit[4]

	type readme struct {
		DownloadUrl string `json:"download_url"`
	}

	readmeUrl := "https://api.github.com/repos/" + owner + "/" + repo + "/readme"
	res, err := http.Get(readmeUrl)
	if err != nil {
		return nil, err
	}
	if res.Body != nil {
		defer res.Body.Close()
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
		resp, err := http.Get(result.DownloadUrl)
		if err != nil {
			return nil, err;
		}
		if resp.Body != nil {
			defer res.Body.Close()
		}
		if resp.StatusCode == http.StatusOK {
			return &source{resp.Body, result.DownloadUrl}, nil
		}
	}

	return nil, errors.New("can't find README in GitHub repository")
}
