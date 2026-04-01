//go:build darwin
// +build darwin

package ui

import "path/filepath"

func ignorePatterns(m commonModel) []string {
	patterns := []string{
		filepath.Join(m.cfg.HomeDir, "Library"),
		m.cfg.Gopath,
		"node_modules",
	}
	if !m.cfg.ShowHiddenFiles {
		patterns = append(patterns, ".*")
	}
	return patterns
}
