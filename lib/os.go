package lib

import (
	"errors"
	"os"
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

// NoSuchFile return true is file name does not exists
func NoSuchFile(name string) bool {
	if _, err := os.Stat(name); errors.Is(err, os.ErrNotExist) {
		return true
	}
	return false
}

// The fileSize returns size of file or zero
func FileSize(name string) int64 {
	fi, err := os.Stat(name)
	if err != nil {
		return 0

	}
	return fi.Size()
}
