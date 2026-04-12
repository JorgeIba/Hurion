package providers

import (
	"fmt"
	"os"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/discorderr"
	"github.com/bwmarrin/discordgo"
)

type DiscordGoSession interface {
	GuildMember(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error)
}

// DiscordValidator focuses on permission checks.
//go:generate go tool mockgen -destination=mock_providers/mock_validator.go -package=mock_providers . DiscordValidator
type DiscordValidator interface {
	IsAdmin(guildID, userID string) (bool, error)
	IsMentor(guildID, userID string) (bool, error)
}

// DiscordFetcher focuses on data retrieval from Discord.
//go:generate go tool mockgen -destination=mock_providers/mock_fetcher.go -package=mock_providers . DiscordFetcher
type DiscordFetcher interface {
	// Future methods like FetchNicknames go here
}

// DiscordProvider combines all Discord-related capabilities.
type DiscordProvider interface {
	DiscordValidator
	DiscordFetcher
}

type discordProvider struct {
	session      DiscordGoSession
	adminRoleID  string
	mentorRoleID string
}

func NewProviderDiscord(s DiscordGoSession) DiscordProvider {
	return &discordProvider{
		session:      s,
		adminRoleID:  os.Getenv("DISCORD_ADMIN_ROLE_ID"),
		mentorRoleID: os.Getenv("DISCORD_MENTOR_ROLE_ID"),
	}
}

func (provider *discordProvider) hasRole(guildID, userID, roleID string) (bool, error) {
	if roleID == "" {
		return false, discorderr.NewInternalError("role ID is not configured", fmt.Errorf("empty role ID"))
	}
	
	member, err := provider.session.GuildMember(guildID, userID)
	if err != nil {
		return false, discorderr.NewInternalError("could not get member", err)
	}
	
	for _, role := range member.Roles {
		if role == roleID {
			return true, nil
		}
	}

	return false, nil
}

func (provider *discordProvider) IsAdmin(guildID, userID string) (bool, error) {
	return provider.hasRole(guildID, userID, provider.adminRoleID)
}

func (provider *discordProvider) IsMentor(guildID, userID string) (bool, error) {
	return provider.hasRole(guildID, userID, provider.mentorRoleID)
}
