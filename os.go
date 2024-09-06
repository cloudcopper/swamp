package main

import "os"

func isDirectoryExist(name string) bool {
	s, err := os.Stat(name)
	if err != nil {
		return false
	}
	if !s.IsDir() {
		return false
	}

	return true
}

// The fileSize returns size of file or zero
func fileSize(name string) int64 {
	fi, err := os.Stat(name)
	if err != nil {
		return 0

	}
	return fi.Size()
}
