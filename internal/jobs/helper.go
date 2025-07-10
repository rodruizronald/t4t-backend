package jobs

import (
	"strings"
	"unicode"
)

// containsSuspiciousPatterns checks for potentially malicious input patterns
func containsSuspiciousPatterns(query string) bool {
	// Check for excessive special characters that might indicate injection attempts
	specialCharCount := 0
	for _, char := range query {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && !unicode.IsSpace(char) &&
			char != '-' && char != '_' && char != '.' && char != '+' && char != '#' {
			specialCharCount++
		}
	}

	// If more than 20% of characters are special characters, flag as suspicious
	if float64(specialCharCount)/float64(len(query)) > 0.2 {
		return true
	}

	// Check for SQL-like patterns (even though we use parameterized queries)
	suspiciousPatterns := []string{
		"--", "/*", "*/", "xp_", "sp_", "exec", "execute", "union", "select",
		"insert", "update", "delete", "drop", "create", "alter",
	}

	lowerQuery := strings.ToLower(query)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerQuery, pattern) {
			return true
		}
	}

	return false
}
