package kdlib

import (
	"regexp"
	"strings"
)

func IsDomainNameValid(domain string) bool {
	if len(domain) > 253 {
		return false
	}

	parts := strings.Split(domain, ".")

	// Checks for a valid label (alphanumeric and hyphen, but not beginning or ending with a hyphen)
	// according to the specifications.
	labelRegExp := regexp.MustCompile(`^[a-zA-Z0-9_]([a-zA-Z0-9\-_]{0,61}[a-zA-Z0-9_])?$`)

	for _, part := range parts {
		if !labelRegExp.MatchString(part) {
			return false
		}
	}

	return true
}
