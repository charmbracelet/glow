// +build !darwin

package ui

// Whether or not we should ignore the given path
func ignorePath(m model, p string) bool {
	if isDotFileOrDir(m.cwd, p) {
		return true
	}
	return pathIsChild(m.cfg.Gopath, p)
}
