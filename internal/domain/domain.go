package domain

import (
	"regexp"
	"strings"
)

var domainPattern = regexp.MustCompile(`(?i)^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+(?:xn--[a-z0-9-]{2,59}|[a-z0-9]{2,63})$`)

func Normalize(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.Trim(trimmed, "`'\"[](){}<>|")
	trimmed = strings.TrimSuffix(trimmed, ".")
	return strings.ToLower(trimmed)
}

func IsValid(value string) bool {
	if len(value) == 0 || len(value) > 253 {
		return false
	}

	return domainPattern.MatchString(value)
}
