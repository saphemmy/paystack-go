package testutil

import (
	"os"
	"path/filepath"
	"runtime"
)

// repoRoot walks up from start looking for a go.mod. Returns "" if none is
// found before reaching the filesystem root.
func repoRoot(start string) string {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// moduleRootPath is resolved at package init by walking up from this file's
// directory. If it cannot be determined (e.g. the file has been stripped in a
// stripped binary), it falls back to the working directory.
var moduleRootPath = func() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	if root := repoRoot(filepath.Dir(file)); root != "" {
		return root
	}
	return "."
}()

// moduleRoot returns the directory containing the go.mod for this module.
func moduleRoot() string { return moduleRootPath }
