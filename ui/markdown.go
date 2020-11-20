package ui

import (
	"log"
	"strings"

	"github.com/charmbracelet/charm"
)

// markdownType allows us to differentiate between the types of markdown
// documents we're dealing with.
type markdownType int

const (
	stashedMarkdown markdownType = iota
	newsMarkdown
	localMarkdown
	convertedMarkdown // used to be local, now its stashed
)

// markdown wraps charm.Markdown.
type markdown struct {
	markdownType markdownType

	// Full path of a local markdown file. Only relevant to local documents and
	// those that have been stashed in this session.
	localPath string

	// Value we filter against. This exists so that we can maintain positions
	// of filtered items if notes are edited while a filter is active. This
	// field is ephemeral, and should only be referenced during filtering.
	filterValue string

	charm.Markdown
}

func (m *markdown) buildFilterValue() {
	note, err := normalize(m.Note)
	if err != nil {
		if debug {
			log.Printf("error normalizing '%s': %v", m.Note, err)
		}
		m.filterValue = m.Note
	}

	if m.markdownType == newsMarkdown {
		m.filterValue = "News: " + note
		return
	}

	m.filterValue = note
}

// sortAsLocal returns whether or not this markdown should be sorted as though
// it's a local markdown document.
func (m markdown) sortAsLocal() bool {
	return m.markdownType == localMarkdown || m.markdownType == convertedMarkdown
}

// Sort documents with local files first, then by date.
type markdownsByLocalFirst []*markdown

func (m markdownsByLocalFirst) Len() int      { return len(m) }
func (m markdownsByLocalFirst) Swap(i, j int) { m[i], m[j] = m[j], m[i] }
func (m markdownsByLocalFirst) Less(i, j int) bool {
	iIsLocal := m[i].sortAsLocal()
	jIsLocal := m[j].sortAsLocal()

	// Local files (and files that used to be local) come first
	if iIsLocal && !jIsLocal {
		return true
	}
	if !iIsLocal && jIsLocal {
		return false
	}

	// If both are local files, sort by filename. Note that we should never
	// hit equality here since two files can't have the same path.
	if iIsLocal && jIsLocal {
		return strings.Compare(m[i].localPath, m[j].localPath) == -1
	}

	// Neither are local files so sort by date descending
	if !m[i].CreatedAt.Equal(m[j].CreatedAt) {
		return m[i].CreatedAt.After(m[j].CreatedAt)
	}

	// If the timestamps also match, sort by ID.
	return m[i].ID > m[j].ID
}
