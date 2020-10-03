package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// gitLabRepo contains information about a GitLab repo
type gitLabRepo struct {
	Branch    string `json:"default_branch"`
	ReadmeURL string `json:"readme_url"`
}

// isGitLabURL tests a string to determine if it is a well-structured GitLab URL
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

// findGitLabREADME tries to find the correct README filename in a repository
func findGitLabREADME(repoURL string) (*source, error) {
	user, repo, err := userAndRepoFromURL(repoURL)
	if err != nil {
		return nil, err
	}

	apiRepoURL := fmt.Sprintf("https://gitlab.com/api/v4/projects/%s%s%s", user, "%2F", repo)

	resp, err := http.Get(apiRepoURL)
	if err != nil {
		return nil, err
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	repoInfo := gitLabRepo{}

	if err := json.Unmarshal([]byte(string(content)), &repoInfo); err != nil {
		return nil, err
	}

	file := strings.Split(repoInfo.ReadmeURL, "/")[8]

	rawReadmeURL := fmt.Sprintf("%s/-/raw/%s/%s", repoURL, repoInfo.Branch, file)

	resp, err = http.Get(rawReadmeURL)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		return &source{
			reader: resp.Body,
			URL:    repoURL,
		}, nil
	}

	return nil, errors.New("can't find README in GitLab repository")
}
