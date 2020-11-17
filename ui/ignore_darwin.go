// +build darwin

package ui

import "path/filepath"

func ignorePatterns(m model) []string {
	return []string{
		filepath.Join(m.general.cfg.HomeDir, "Library"),
		m.general.cfg.Gopath,
		"node_modules",
		".*",
	}
}
