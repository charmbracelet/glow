//go:build darwin
// +build darwin

package ui

import "path/filepath"

func ignorePatterns(m commonModel) []string {
	return []string{
		filepath.Join(m.cfg.HomeDir, "Library"),
		m.cfg.Gopath,
		"node_modules",
		".*",
	}
}
