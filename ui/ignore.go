package ui

import (
	"os"
	"strings"
)

// Returns whether or not the given path contains a file or directory starting
// with a dot. This is relative to the current working directory, so if you're
// in a dot directory and browsing files beneath this function won't return
// true every time.
func isDotFileOrDir(cwd, path string) bool {
	p := strings.TrimPrefix(path, cwd)
	for _, v := range strings.Split(p, string(os.PathSeparator)) {
		if len(v) > 0 && v[0] == '.' {
			return true
		}
	}
	return false
}

// Returns whether or not a path is a child of a given path. For example:
//
//      parent := "/usr/local/bin"
//      child := "/usr/local/bin/glow"
//      pathIsChild(parent, child) // true
//
func pathIsChild(parent, child string) bool {
	if len(parent) == 0 || len(child) == 0 {
		return false
	}
	if len(parent) > len(child) {
		return false
	}
	if strings.Compare(parent, child[:len(parent)]) == 0 {
		return true
	}
	return false
}
