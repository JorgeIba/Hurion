package common

import (
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/common"
	"github.com/bwmarrin/discordgo"
)

// ResolveUserID extracts a userID from a mention or returns the raw input as fallback
func ResolveUserID(param string) string {
	if id, ok := common.ExtractUserIDFromMention(param); ok {
		return id
	}
	return param
}

// GetUserLabel returns a human-readable label for a Discord user, prioritizing server nickname
func GetUserLabel(user *discordgo.User, member *discordgo.Member) string {
	if member != nil && member.Nick != "" {
		return member.Nick
	}
	if user.GlobalName != "" {
		return user.GlobalName
	}
	if user.Username != "" {
		return user.Username
	}
	return fmt.Sprintf("<@%s>", user.ID)
}
