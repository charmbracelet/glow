package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// isGitHubURL tests a string to determine if it is a well-structured GitHub URL
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

// findGitHubREADME tries to find the correct README filename in a repository
func findGitHubREADME(repoURL string) (*source, error) {
	user, repo, err := userAndRepoFromURL(repoURL)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}

	apiReadmeURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/readme", user, repo)

	req, err := http.NewRequest("GET", apiReadmeURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/vnd.github.v3.raw")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		return &source{
			reader: resp.Body,
			URL:    repoURL,
		}, nil
	}

	// if GitHub API rate limit is exceeded try to guess the readme path
	if resp.StatusCode == http.StatusForbidden {
		for _, r := range readmeNames {
			rawReadmeURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/master/%s", user, repo, r)

			resp, err := http.Get(rawReadmeURL)
			if err != nil {
				return nil, err
			}

			if resp.StatusCode == http.StatusOK {
				fmt.Println(rawReadmeURL)
				return &source{
					reader: resp.Body,
					URL:    repoURL,
				}, nil
			}
		}
	}

	return nil, errors.New("can't find README in GitHub repository")
}
