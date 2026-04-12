package providers

import (
	"fmt"

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

type hasRoleArgs struct {
	guildID string
	userID  string
	roleID  string
}

func NewProviderDiscord(s DiscordGoSession, adminRoleID, mentorRoleID string) DiscordProvider {
	return &discordProvider{
		session:      s,
		adminRoleID:  adminRoleID,
		mentorRoleID: mentorRoleID,
	}
}

func (provider *discordProvider) hasRole(args hasRoleArgs) (bool, error) {
	if provider.session == nil {
		return false, discorderr.NewInternalError("discord session not configured", fmt.Errorf("nil session"))
	}

	if args.roleID == "" {
		return false, discorderr.NewInternalError("role ID is not configured", fmt.Errorf("empty role ID"))
	}
	
	member, err := provider.session.GuildMember(args.guildID, args.userID)
	if err != nil {
		return false, discorderr.NewInternalError("could not get member", err)
	}
	
	for _, role := range member.Roles {
		if role == args.roleID {
			return true, nil
		}
	}

	return false, nil
}

func (provider *discordProvider) IsAdmin(guildID, userID string) (bool, error) {
	return provider.hasRole(hasRoleArgs{
		guildID: guildID,
		userID:  userID,
		roleID:  provider.adminRoleID,
	})
}

func (provider *discordProvider) IsMentor(guildID, userID string) (bool, error) {
	return provider.hasRole(hasRoleArgs{
		guildID: guildID,
		userID:  userID,
		roleID:  provider.mentorRoleID,
	})
}
