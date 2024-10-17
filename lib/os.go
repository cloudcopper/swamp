package lib

import (
	"errors"
	"os"

	"github.com/spf13/afero"
)

func IsDirectoryExist(name string) bool {
	s, err := os.Stat(name)
	if err != nil {
		return false
	}
	if !s.IsDir() {
		return false
	}

	return true
}

// NoSuchFile return true if file name does not exists
func NoSuchFile(fs afero.Fs, name string) bool {
	if _, err := fs.Stat(name); errors.Is(err, os.ErrNotExist) {
		return true
	}
	return false
}

// The fileSize returns size of file or zero
func FileSize(fs afero.Fs, name string) int64 {
	fi, err := fs.Stat(name)
	if err != nil {
		return 0

	}
	return fi.Size()
}
