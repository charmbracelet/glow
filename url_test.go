package main

import (
	"context"
	"testing"
)

func TestURLParser(t *testing.T) {
	for path, url := range map[string]string{
		"github.com/charmbracelet/glow":             "https://raw.githubusercontent.com/charmbracelet/glow/master/README.md",
		"github://charmbracelet/glow":               "https://raw.githubusercontent.com/charmbracelet/glow/master/README.md",
		"github://caarlos0/dotfiles.fish":           "https://raw.githubusercontent.com/caarlos0/dotfiles.fish/main/README.md",
		"github://tj/git-extras":                    "https://raw.githubusercontent.com/tj/git-extras/main/Readme.md",
		"https://github.com/goreleaser/nfpm":        "https://raw.githubusercontent.com/goreleaser/nfpm/main/README.md",
		"gitlab.com/caarlos0/test":                  "https://gitlab.com/caarlos0/test/-/raw/master/README.md",
		"gitlab://caarlos0/test":                    "https://gitlab.com/caarlos0/test/-/raw/master/README.md",
		"https://gitlab.com/terrakok/gitlab-client": "https://gitlab.com/terrakok/gitlab-client/-/raw/develop/Readme.md",
	} {
		t.Run(path, func(t *testing.T) {
			t.Skip("test uses network, sometimes fails for no reason")
			got, err := readmeURL(context.Background(), path)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got == nil {
				t.Fatalf("should not be nil")
			}
			if url != got.URL {
				t.Errorf("expected url for %s to be %s, was %s", path, url, got.URL)
			}
		})
	}
}
