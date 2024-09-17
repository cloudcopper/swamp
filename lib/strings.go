package lib

import (
	"regexp"
	"strings"
	"unicode"
)

func LeadingDigits(s string) string {
	for i, r := range s {
		if !unicode.IsDigit(r) {
			return s[:i]
		}
	}
	return s
}

var reValidID = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9-_.]*$")

func IsValidID(s string) bool {
	if !reValidID.MatchString(s) {
		return false
	}
	if strings.Contains(s, "..") {
		return false
	}
	return true
}
