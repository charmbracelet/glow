package ui

import (
	"cmp"
	"slices"
	"time"
)

type SortType int
type SortOrder int

const (
	SortByNote SortType = iota
	SortByDate
	SortByTitle
)

const (
	SortAscending SortOrder = iota
	SortDescending
)

type SortState struct {
	Type  SortType
	Order SortOrder
}

func (s *SortState) Toggle(newType SortType) {
	if s.Type == newType {
		// Toggle order if same type
		if s.Order == SortAscending {
			s.Order = SortDescending
		} else {
			s.Order = SortAscending
		}
	} else {
		// Set new type with default ascending order
		s.Type = newType
		s.Order = SortAscending
	}
}

func sortMarkdowns(mds []*markdown, state SortState) {
	slices.SortStableFunc(mds, func(a, b *markdown) int {
		var comparison int

		switch state.Type {
		case SortByDate:
			comparison = compareTime(a.Modtime, b.Modtime)
		case SortByTitle:
			comparison = cmp.Compare(a.Note, b.Note)
		default: // SortByNote
			comparison = cmp.Compare(a.Note, b.Note)
		}

		if state.Order == SortDescending {
			comparison = -comparison
		}

		return comparison
	})
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
