package lib

import "unicode"

func LeadingDigits(s string) string {
	for i, r := range s {
		if !unicode.IsDigit(r) {
			return s[:i]
		}
	}
	return s
}
