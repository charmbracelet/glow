package main

import (
	"encoding/json"
	"errors"
	"io"
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

// findGitLabREADME tries to find the correct README filename in a repository using GitLab API.
func findGitLabREADME(s string) (*source, error) {
	sSplit := strings.Split(s, "/")
	owner, repo := sSplit[3], sSplit[4]

	projectPath := url.QueryEscape(owner + "/" + repo)

	type readme struct {
		ReadmeUrl string `json:"readme_url"`
	}

	apiURL := "https://gitlab.com/api/v4/projects/" + projectPath

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

	readmeRawUrl := strings.Replace(result.ReadmeUrl, "blob", "raw", -1)

	if res.StatusCode == http.StatusOK {
		// nolint:bodyclose
		// it is closed on the caller
		resp, err := http.Get(readmeRawUrl)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusOK {
			return &source{resp.Body, readmeRawUrl}, nil
		}
	}

	return nil, errors.New("can't find README in GitLab repository")
}
