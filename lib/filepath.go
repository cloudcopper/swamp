package lib

import (
	"os"
	"path/filepath"
	"strings"
)

// GetFirstSubdir returns first directory name after root
// Example:
// input is '/mnt/input/project'
// if path is '/mnt/input/project/1234.crc' then return is ”
// if path is '/mnt/input/project/rel-4.2.2/1234.crc' then return is 'rel-4.2.2'
func GetFirstSubdir(root, path string) string {
	Assert(strings.HasPrefix(path, root))
	a := strings.Split(strings.TrimLeft(strings.TrimPrefix(path, root), string(os.PathSeparator)), string(os.PathSeparator))
	Assert(len(a) >= 1)
	Assert(a[0] != "")
	dir := a[0]
	if len(a) <= 1 {
		dir = ""
	}

	return dir
}

// IsAbs covers problem of  filepath.IsAbs which only checks
// first element of path and allows .. inside.
// The filepath.Abs meanwhile does filepath.Clean.
// So this function returns true, if filepath.Abs returns very same value
func IsAbs(path string) bool {
	if abs, err := filepath.Abs(path); err != nil || abs != path {
		return false
	}

	return true
}
