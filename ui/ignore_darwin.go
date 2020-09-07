// +build darwin

package ui

import "os"

func ignorePatterns(m model) []string {
	return []string{
		m.cfg.HomeDir + string(os.PathSeparator) + "Library",
		m.cfg.Gopath,
		"node_modules",
		".*",
	}
}
