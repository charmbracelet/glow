package ui

import (
	"reflect"
	"testing"
)

func TestDocTypeContains(t *testing.T) {
	d := NewDocTypeSet(LocalDoc)

	if !d.Contains(LocalDoc) {
		t.Error("Contains reported it doesn't contain a value which it absolutely does contain")
	}

	if d.Contains(NewsDoc) {
		t.Error("Contains reported the set contains a value it certainly does not")
	}
}

func TestDocTypeDifference(t *testing.T) {
	original := NewDocTypeSet(LocalDoc, StashedDoc, ConvertedDoc, NewsDoc)
	difference := original.Difference(LocalDoc, NewsDoc)
	expected := NewDocTypeSet(StashedDoc, ConvertedDoc)

	// Make sure the difference operation worked
	if !reflect.DeepEqual(difference, expected) {
		t.Errorf("difference returned %+v; expected %+v", difference, expected)
	}

	// Make sure original set was not mutated
	if reflect.DeepEqual(original, difference) {
		t.Errorf("original set was mutated when it should not have been")
	}
}

func TestDocTypeEquality(t *testing.T) {
	a := NewDocTypeSet(LocalDoc, StashedDoc)
	b := NewDocTypeSet(LocalDoc, StashedDoc)
	c := NewDocTypeSet(LocalDoc)

	if !a.Equals(b) {
		t.Errorf("Equality test failed for %+v and %+v; expected true, got false", a, b)
	}

	if a.Equals(c) {
		t.Errorf("Equality test failed for %+v and %+v; expected false, got true", a, c)
	}
}
