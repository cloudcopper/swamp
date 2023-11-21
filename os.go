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
