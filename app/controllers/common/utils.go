package common

import (
	"fmt"
	"unicode"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/common"
	"github.com/bwmarrin/discordgo"
)

// ParseUserID extracts a userID from a mention or verifies if the param is a raw numeric ID.
// It returns the parsed ID and a boolean indicating if the parsing was successful.
func ParseUserID(param string) (string, bool) {
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
	if name := GetDisplayNameFromMember(member); name != "" {
		return name
	}

	if name := GetDisplayNameFromUser(user); name != "" {
		return name
	}

	if user != nil {
		return fmt.Sprintf("<@%s>", user.ID)
	}

	return "<unknown user>"
}
