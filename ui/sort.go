package ui

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"
)

func sortMarkdowns(mds []*markdown) {
	slices.SortStableFunc(mds, func(a, b *markdown) int {
		if n := cmp.Compare(markdownPathDepth(a.Note), markdownPathDepth(b.Note)); n != 0 {
			return n
		}
		return cmp.Compare(a.Note, b.Note)
	})
}

func markdownPathDepth(path string) int {
	path = filepath.ToSlash(filepath.Clean(path))
	if path == "." {
		return 0
	}
	return strings.Count(path, "/")
}
