package ui

import (
	"cmp"
	"slices"
	"time"
)

type sortOrder bool

const (
	ascending  sortOrder = true
	descending sortOrder = false
)

func sortMarkdowns(mds []*markdown, sortByDate bool, ascending sortOrder) {
	if sortByDate {
		if ascending {
			slices.SortStableFunc(mds, func(a, b *markdown) int {
				return compareTime(a.Modtime, b.Modtime)
			})
		} else {
			slices.SortStableFunc(mds, func(a, b *markdown) int {
				return -compareTime(a.Modtime, b.Modtime)
			})
		}
	} else {
		if ascending {
			slices.SortStableFunc(mds, func(a, b *markdown) int {
				return cmp.Compare(a.Note, b.Note)
			})
		} else {
			slices.SortStableFunc(mds, func(a, b *markdown) int {
				return -cmp.Compare(a.Note, b.Note)
			})
		}
	}
}

func compareTime(a, b time.Time) int {
	switch {
	case a.Before(b):
		return -1
	case a.After(b):
		return 1
	default:
		return 0
	}
}
