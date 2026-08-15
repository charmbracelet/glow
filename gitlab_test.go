package main

import (
	"testing"
)

func TestGitLabRawURLPreservesBlobInProjectName(t *testing.T) {
	const readmeURL = "https://gitlab.com/Sjoerdlab/openblob/-/blob/main/README.md"
	const want = "https://gitlab.com/Sjoerdlab/openblob/-/raw/main/README.md"

	if got := gitLabRawURL(readmeURL); got != want {
		t.Fatalf("expected raw README URL %q, got %q", want, got)
	}
}
