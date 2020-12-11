// +build !darwin

package ui

func ignorePatterns(m model) []string {
	return []string{
		m.common.cfg.Gopath,
		"node_modules",
		".*",
	}
}
