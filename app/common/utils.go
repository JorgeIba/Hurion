package common

import (
	"strings"
)

// ExtractUserIDFromMention extracts the numerical Discord User ID from a mention string.
// Supported formats: <@123...>, <@!123...>
func ExtractUserIDFromMention(mention string) (string, bool) {
	if !strings.HasPrefix(mention, "<@") || !strings.HasSuffix(mention, ">") {
		return "", false
	}
	userID := strings.TrimPrefix(mention, "<@")
	userID = strings.TrimPrefix(userID, "!")
	userID = strings.TrimSuffix(userID, ">")

	if len(userID) == 0 {
		return "", false
	}

	// Discord IDs (Snowflakes) are numeric
	for _, c := range userID {
		if c < '0' || c > '9' {
			return "", false
		}
	}

	return userID, true
}
