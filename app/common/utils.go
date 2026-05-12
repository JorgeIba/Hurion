package common

import (
	"strings"
)

// IsUserMention returns true if the string looks like a Discord user mention.
func IsUserMention(s string) bool {
	// User mentions: <@123...> or <@!123...>
	// Role mentions: <@&123...>
	// We must exclude the '&' prefix.
	if strings.HasPrefix(s, "<@&") {
		return false
	}
	return (strings.HasPrefix(s, "<@") || strings.HasPrefix(s, "<@!")) && strings.HasSuffix(s, ">")
}

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

// ExtractRoleIDFromMention extracts the numerical Discord Role ID from a mention string.
// Supported format: <@&123...>
func ExtractRoleIDFromMention(mention string) (string, bool) {
	if !strings.HasPrefix(mention, "<@&") || !strings.HasSuffix(mention, ">") {
		return "", false
	}
	roleID := strings.TrimPrefix(mention, "<@&")
	roleID = strings.TrimSuffix(roleID, ">")

	if len(roleID) == 0 {
		return "", false
	}

	for _, c := range roleID {
		if c < '0' || c > '9' {
			return "", false
		}
	}

	return roleID, true
}

// ExtractChannelIDFromMention extracts the numerical Discord Channel ID from a mention string.
// Supported format: <#123...>
func ExtractChannelIDFromMention(mention string) (string, bool) {
	if !strings.HasPrefix(mention, "<#") || !strings.HasSuffix(mention, ">") {
		return "", false
	}
	channelID := strings.TrimPrefix(mention, "<#")
	channelID = strings.TrimSuffix(channelID, ">")

	if len(channelID) == 0 {
		return "", false
	}

	for _, c := range channelID {
		if c < '0' || c > '9' {
			return "", false
		}
	}

	return channelID, true
}
