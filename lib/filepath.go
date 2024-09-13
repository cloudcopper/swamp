package lib

import (
	"os"
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
