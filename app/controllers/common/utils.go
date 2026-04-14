package common

import (
	"fmt"
	"unicode"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/common"
	"github.com/bwmarrin/discordgo"
)

// ResolveUserID extracts a userID from a mention or verifies if the param is a raw numeric ID.
// It returns the resolved ID and a boolean indicating if the resolution was successful.
func ResolveUserID(param string) (string, bool) {
	// Try to extract from mention first
	if id, ok := common.ExtractUserIDFromMention(param); ok {
		return id, true
	}

	// If not a mention, verify it's a raw numeric ID
	if len(param) == 0 {
		return "", false
	}

	for _, c := range param {
		if !unicode.IsDigit(c) {
			return "", false
		}
	}

	return param, true
}

// GetUserLabel returns a human-readable label for a Discord user.
func GetUserLabel(user *discordgo.User, member *discordgo.Member) string {
	if member != nil && member.Nick != "" {
		return member.Nick
	}

	if user == nil {
		if member != nil && member.User != nil {
			user = member.User
		} else {
			return "<unknown user>"
		}
	}

	if user.GlobalName != "" {
		return user.GlobalName
	}
	if user.Username != "" {
		return user.Username
	}
	return fmt.Sprintf("<@%s>", user.ID)
}
