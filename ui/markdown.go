package ui

import (
	"log"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/charm"
	"github.com/dustin/go-humanize"
	"github.com/segmentio/ksuid"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// markdown wraps charm.Markdown.
type markdown struct {
	markdownType DocType

	// Local identifier. This allows us to precisely determine the stashed
	// state of a markdown, regardless of whether it exists locally or on the
	// network.
	localID ksuid.KSUID

	// Full path of a local markdown file. Only relevant to local documents and
	// those that have been stashed in this session.
	localPath string

	// Value we filter against. This exists so that we can maintain positions
	// of filtered items if notes are edited while a filter is active. This
	// field is ephemeral, and should only be referenced during filtering.
	filterValue string

	charm.Markdown
}

func (m *markdown) generateLocalID() {
	if m.localID.IsNil() {
		m.localID = ksuid.New()
	}
}

// Generate the value we're doing to filter against.
func (m *markdown) buildFilterValue() {
	note, err := normalize(m.Note)
	if err != nil {
		if debug {
			log.Printf("error normalizing '%s': %v", m.Note, err)
		}
		m.filterValue = m.Note
	}

	m.filterValue = note
}

// sortAsLocal returns whether or not this markdown should be sorted as though
// it's a local markdown document.
func (m markdown) sortAsLocal() bool {
	return m.markdownType == LocalDoc || m.markdownType == ConvertedDoc
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

func (m markdown) relativeTime() string {
	return relativeTime(m.CreatedAt)
}

// Normalize text to aid in the filtering process. In particular, we remove
// diacritics, "ö" becomes "o". Note that Mn is the unicode key for nonspacing
// marks.
func normalize(in string) (string, error) {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, err := transform.String(t, in)
	return out, err
}

// wrapMarkdowns wraps a *charm.Markdown with a *markdown in order to add some
// extra metadata.
func wrapMarkdowns(t DocType, md []*charm.Markdown) (m []*markdown) {
	for _, v := range md {
		m = append(m, &markdown{
			markdownType: t,
			Markdown:     *v,
		})
	}
	return m
}

// Return the time in a human-readable format relative to the current time.
func relativeTime(then time.Time) string {
	now := time.Now()
	ago := now.Sub(then)
	if ago < time.Minute {
		return "just now"
	} else if ago < humanize.Week {
		return humanize.CustomRelTime(then, now, "ago", "from now", magnitudes)
	}
	return then.Format("02 Jan 2006 15:04 MST")
}

// Magnitudes for relative time.
var magnitudes = []humanize.RelTimeMagnitude{
	{D: time.Second, Format: "now", DivBy: time.Second},
	{D: 2 * time.Second, Format: "1 second %s", DivBy: 1},
	{D: time.Minute, Format: "%d seconds %s", DivBy: time.Second},
	{D: 2 * time.Minute, Format: "1 minute %s", DivBy: 1},
	{D: time.Hour, Format: "%d minutes %s", DivBy: time.Minute},
	{D: 2 * time.Hour, Format: "1 hour %s", DivBy: 1},
	{D: humanize.Day, Format: "%d hours %s", DivBy: time.Hour},
	{D: 2 * humanize.Day, Format: "1 day %s", DivBy: 1},
	{D: humanize.Week, Format: "%d days %s", DivBy: humanize.Day},
	{D: 2 * humanize.Week, Format: "1 week %s", DivBy: 1},
	{D: humanize.Month, Format: "%d weeks %s", DivBy: humanize.Week},
	{D: 2 * humanize.Month, Format: "1 month %s", DivBy: 1},
	{D: humanize.Year, Format: "%d months %s", DivBy: humanize.Month},
	{D: 18 * humanize.Month, Format: "1 year %s", DivBy: 1},
	{D: 2 * humanize.Year, Format: "2 years %s", DivBy: 1},
	{D: humanize.LongTime, Format: "%d years %s", DivBy: humanize.Year},
	{D: math.MaxInt64, Format: "a long while %s", DivBy: 1},
}
