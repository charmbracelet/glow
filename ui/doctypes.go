package ui

// DocType represents a type of markdown document.
type DocType int

// Available document types.
const (
	NoDocType DocType = iota
	LocalDoc
	StashedDoc
	ConvertedDoc
	NewsDoc
)

func (d DocType) String() string {
	return [...]string{
		"none",
		"local",
		"stashed",
		"converted",
		"news",
	}[d]
}

// DocTypeSet is a set (in the mathematic sense) of document types.
type DocTypeSet map[DocType]struct{}

// NewDocTypeSet returns a set of document types.
func NewDocTypeSet(t ...DocType) DocTypeSet {
	d := DocTypeSet(make(map[DocType]struct{}))
	if len(t) > 0 {
		d.Add(t...)
	}
	return d
}

// Add adds a document type of the set.
func (d *DocTypeSet) Add(t ...DocType) int {
	for _, v := range t {
		(*d)[v] = struct{}{}
	}
	return len(*d)
}

// Contains returns whether or not the set contains the given DocTypes.
func (d DocTypeSet) Contains(m ...DocType) bool {
	matches := 0
	for _, t := range m {
		if _, found := d[t]; found {
			matches++
		}
	}
	return matches == len(m)
}

// Difference return a DocumentType set that does not contain the given types.
func (d DocTypeSet) Difference(t ...DocType) DocTypeSet {
	c := copyDocumentTypes(d)
	for k := range c {
		for _, docType := range t {
			if k == docType {
				delete(c, k)
				break
			}
		}
	}
	return c
}

// Equals returns whether or not the two sets are equal.
func (d DocTypeSet) Equals(other DocTypeSet) bool {
	return d.Contains(other.AsSlice()...) && len(d) == len(other)
}

// AsSlice returns the set as a slice of document types.
func (d DocTypeSet) AsSlice() (agg []DocType) {
	for k := range d {
		agg = append(agg, k)
	}
	return
}

// Return a copy of the given DoctTypes map.
func copyDocumentTypes(d DocTypeSet) DocTypeSet {
	c := make(map[DocType]struct{})
	for k, v := range d {
		c[k] = v
	}
	return c
}
