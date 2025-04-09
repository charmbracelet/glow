package main

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
)

const (
	protoGithub = "github://"
	protoGitlab = "gitlab://"
	protoHTTPS  = "https://"
)

var (
	githubURL *url.URL
	gitlabURL *url.URL
	urlsOnce  sync.Once
)

func init() {
	urlsOnce.Do(func() {
		githubURL, _ = url.Parse("https://github.com")
		gitlabURL, _ = url.Parse("https://gitlab.com")
	})
}

func readmeURL(path string) (*source, error) {
	switch {
	case strings.HasPrefix(path, protoGithub):
		if u := githubReadmeURL(path); u != nil {
			return readmeURL(u.String())
		}
		return nil, nil
	case strings.HasPrefix(path, protoGitlab):
		if u := gitlabReadmeURL(path); u != nil {
			return readmeURL(u.String())
		}
		return nil, nil
	}

	if !strings.HasPrefix(path, protoHTTPS) {
		path = protoHTTPS + path
	}
	u, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("unable to parse url: %w", err)
	}

	switch {
	case u.Hostname() == githubURL.Hostname():
		return findGitHubREADME(u)
	case u.Hostname() == gitlabURL.Hostname():
		return findGitLabREADME(u)
	}

	return nil, nil
}

func githubReadmeURL(path string) *url.URL {
	path = strings.TrimPrefix(path, protoGithub)
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		// custom hostnames are not supported yet
		return nil
	}
	u, _ := url.Parse(githubURL.String())
	return u.JoinPath(path)
}

func gitlabReadmeURL(path string) *url.URL {
	path = strings.TrimPrefix(path, protoGitlab)
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		// custom hostnames are not supported yet
		return nil
	}
	u, _ := url.Parse(gitlabURL.String())
	return u.JoinPath(path)
}

func isURL(path string) bool {
	_, err := url.ParseRequestURI(path)
	return err == nil && strings.Contains(path, "://")
}
