package random

import (
	"path/filepath"
	"strings"
)

// Filepath returns random path up to max directory
func Filepath(n int) string {
	a := []string{}
	n = Value([]int{0, n})
	for x := 0; x < n; x++ {
		a = append(a, strings.ReplaceAll(Words([]int{1, 3}), " ", "_"))
	}
	return strings.Join(a, string(filepath.Separator))
}

func FileName(n int) string {
	filename := strings.ReplaceAll(Words([]int{1, n}), " ", "_") + "." + Element([]string{"bin", "txt", "srec", "jar", "tar.gz", "html", "iso", "wad"})
	return filename
}
