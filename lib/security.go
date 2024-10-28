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

func IsKeyBlacklisted(key string) bool {
	list := []string{}

	if strings.HasPrefix(key, "_") {
		return true
	}

	for _, term := range list {
		if strings.Contains(key, term) {
			return true
		}
	}

	return false
}

func IsKeyValueBlacklisted(key string) bool {
	list := []string{
		"PASSWORD",
		"SECRET",
	}

	for _, term := range list {
		if strings.Contains(key, term) {
			return true
		}
	}

	return false
}
