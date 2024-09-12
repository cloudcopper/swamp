package lib

import "strings"

func IsSecureFileName(name string) bool {
	if strings.Contains(name, "..") || strings.Contains(name, ":") {
		return false
	}
	return true
}
