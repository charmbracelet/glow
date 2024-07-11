package ui

import (
	"cmp"
	"slices"
)

func sortMarkdowns(mds []*markdown) {
	slices.SortStableFunc(mds, func(a, b *markdown) int {
		return cmp.Compare(a.Note, b.Note)
	})
}
