package ui

import "regexp"

// findMatches returns the byte-offset ranges of every case-insensitive,
// literal occurrence of query within content, in the [][]int{{start, end}, ...}
// format expected by viewport.Model.SetHighlights. Returns nil if query is
// empty or there are no matches.
func findMatches(content, query string) [][]int {
	if query == "" {
		return nil
	}
	re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(query))
	return re.FindAllStringIndex(content, -1)
}
