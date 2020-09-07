// +build darwin

package ui

import "os"

func ignorePath(m model, p string) bool {
	if isDotFileOrDir(m.cwd, p) {
		return true
	}
	if pathIsChild(m.cfg.Gopath, p) {
		return true
	}

	// Look for ~/Library on macOS
	macOSLibraryPath := m.cfg.HomeDir + string(os.PathSeparator) + "Library"
	return pathIsChild(macOSLibraryPath, p)
}
