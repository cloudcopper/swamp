package main

import "strings"

func isSecureFileName(name string) bool {
	if strings.Contains(name, "..") || strings.Contains(name, ":") {
		return false
	}
	return true
}
