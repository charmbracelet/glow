// +build darwin

package ui

import "path/filepath"

func ignorePatterns(m model) []string {
	return []string{
		filepath.Join(m.common.cfg.HomeDir, "Library"),
		m.common.cfg.Gopath,
		"node_modules",
		".*",
	}
}
