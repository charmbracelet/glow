// +build !darwin

package ui

func ignorePatterns(m model) []string {
	return []string{
		m.cfg.Gopath,
		"node_modules"
		".*",
	}
}
