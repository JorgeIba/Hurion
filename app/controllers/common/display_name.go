package common

import (
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
	"github.com/bwmarrin/discordgo"
)

// ResolveDisplayName resolves the display name for a userID.
// It first checks the Discord memory state cache, then falls back to the DB cache.
// If neither is found, it returns the raw userID.
func ResolveDisplayName(c *context.ContextMessage, guildID, userID string) string {
	if guildID == "" || userID == "" {
		return userID
	}

	// 1. Try Discord In-Memory State Cache
	if c.Session != nil && c.Session.State != nil {
		member, err := c.Session.State.Member(guildID, userID)
		if err == nil {
			return GetDisplayNameFromMember(member)
		}
	}

	// 2. Try DB Cache
	if c.Services != nil && c.Services.User != nil {
		user, err := c.Services.User.GetUserByUserID(userID)
		if err == nil && user.DisplayName != "" {
			return user.DisplayName
		}
	}

	// 3. Fallback
	return userID
}

// ResolveDisplayNames resolves display names for a batch of userIDs.
func ResolveDisplayNames(c *context.ContextMessage, guildID string, userIDs []string) map[string]string {
	resolved := make(map[string]string)
	for _, userID := range userIDs {
		resolved[userID] = ResolveDisplayName(c, guildID, userID)
	}
	return resolved
}

// ResolveScoreDisplayNames resolves display names for all users in a Score object.
func ResolveScoreDisplayNames(c *context.ContextMessage, guildID string, score *service_model.Score) {
	if score == nil {
		return
	}
	for i := range score.UserScores {
		score.UserScores[i].User.DisplayName = ResolveDisplayName(c, guildID, score.UserScores[i].User.UserID)
	}
}

// ResolveUsersDisplayNames resolves display names for a list of User objects.
func ResolveUsersDisplayNames(c *context.ContextMessage, guildID string, users []service_model.User) {
	for i := range users {
		users[i].DisplayName = ResolveDisplayName(c, guildID, users[i].UserID)
	}
}

// MapMemberToUser returns a service_model.User with the display name resolved from the Member object.
func MapMemberToUser(member *discordgo.Member) service_model.User {
	if member == nil || member.User == nil {
		return service_model.User{}
	}

	return service_model.User{
		UserID:      member.User.ID,
		DisplayName: GetDisplayNameFromMember(member),
	}
}

// MapDiscordUserToUser returns a service_model.User with the display name resolved from the User object.
func MapDiscordUserToUser(user *discordgo.User) service_model.User {
	if user == nil {
		return service_model.User{}
	}

	return service_model.User{
		UserID:      user.ID,
		DisplayName: GetDisplayNameFromUser(user),
	}
}

// GetDisplayNameFromMember extracts the best display name from a Discord member.
func GetDisplayNameFromMember(member *discordgo.Member) string {
	if member == nil {
		return ""
	}

	if member.Nick != "" {
		return member.Nick
	}

	return GetDisplayNameFromUser(member.User)
}

// GetDisplayNameFromUser extracts the best display name from a Discord user.
func GetDisplayNameFromUser(user *discordgo.User) string {
	if user == nil {
		return ""
	}

	if user.GlobalName != "" {
		return user.GlobalName
	}

	return user.Username
}
