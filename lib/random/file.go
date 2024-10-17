package random

import (
	"path/filepath"
	"strings"
)

// Filepath returns random path up to max directory
func Filepath(max int) string {
	a := []string{}
	max = Value([]int{0, max})
	for x := 0; x < max; x++ {
		a = append(a, strings.ReplaceAll(Words([]int{1, 3}), " ", "_"))
	}
	return strings.Join(a, string(filepath.Separator))
}

func FileName(max int) string {
	filename := strings.ReplaceAll(Words([]int{1, max}), " ", "_") + "." + Element([]string{"bin", "txt", "srec", "jar", "tar.gz", "html", "iso", "wad"})
	return filename
}
