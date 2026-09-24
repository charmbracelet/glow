package ui

import "testing"

func TestSortMarkdownsOrdersByDirectoryDepth(t *testing.T) {
	mds := []*markdown{
		{Note: "docs/reference/api.md"},
		{Note: "docs/guide.md"},
		{Note: "README.md"},
		{Note: "CONTRIBUTING.md"},
		{Note: "docs/reference/cli.md"},
	}

	sortMarkdowns(mds)

	got := make([]string, 0, len(mds))
	for _, md := range mds {
		got = append(got, md.Note)
	}

	want := []string{
		"CONTRIBUTING.md",
		"README.md",
		"docs/guide.md",
		"docs/reference/api.md",
		"docs/reference/cli.md",
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}
