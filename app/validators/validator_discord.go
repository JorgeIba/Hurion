package validators

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type DiscordGoSession interface {
	GuildMember(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error)
}

type ValidatorDiscord struct {
	session DiscordGoSession
}

func NewValidatorDiscord(s DiscordGoSession) *ValidatorDiscord {
	return &ValidatorDiscord{session: s}
}

const (
	ADMIN_ROLE  = "690200603871412245"
	MENTOR_ROLE = "689627803624407041"
)

func (validator *ValidatorDiscord) hasRole(guildID, userID, roleID string) (bool, error) {
	member, err := validator.session.GuildMember(guildID, userID)
	if err != nil {
		return false, fmt.Errorf("could not get member: %v", err)
	}

	for _, role := range member.Roles {
		if role == roleID {
			return true, nil
		}
	}

	return false, nil
}

func (validator *ValidatorDiscord) IsAdmin(guildID, userID string) (bool, error) {
	return validator.hasRole(guildID, userID, ADMIN_ROLE)
}

func (validator *ValidatorDiscord) IsMentor(guildID, userID string) (bool, error) {
	return validator.hasRole(guildID, userID, MENTOR_ROLE)
}
