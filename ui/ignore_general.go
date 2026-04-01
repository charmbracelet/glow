//go:build !darwin
// +build !darwin

package ui

func ignorePatterns(m commonModel) []string {
	patterns := []string{
		m.cfg.Gopath,
		"node_modules",
	}
	if !m.cfg.ShowHiddenFiles {
		patterns = append(patterns, ".*")
	}
	return patterns
}
