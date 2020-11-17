// +build !darwin

package ui

func ignorePatterns(m model) []string {
	return []string{
		m.general.cfg.Gopath,
		"node_modules",
		".*",
	}
}
