package lib

import "strings"

// IsSecureFileName checking the file name does not have some hacks in it
// TODO Is ":" good one to block some windows hacks?
func IsSecureFileName(name string) bool {
	if strings.Contains(name, "..") || strings.Contains(name, "./") || strings.Contains(name, ":") {
		return false
	}
	return true
}
