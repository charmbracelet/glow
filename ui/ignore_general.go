//go:build !darwin
// +build !darwin

package ui

func ignorePatterns(m commonModel) []string {
	return []string{
		m.cfg.Gopath,
		"node_modules",
		".*",
	}
}
