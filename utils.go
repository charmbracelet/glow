package main

import (
	"errors"
	"net/url"
	"strings"
)

func userAndRepoFromURL(repoURL string) (string, string, error) {
	u, err := url.ParseRequestURI(repoURL)
	if err != nil {
		return "", "", err
	}

	pathSplit := strings.Split(u.Path, "/")

	if len(pathSplit) != 3 || pathSplit[2] == "" {
		return "", "", errors.New("Invalid URL format")
	}

	return pathSplit[1], pathSplit[2], nil
}
