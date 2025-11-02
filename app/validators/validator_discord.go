package validators

import (
	"fmt"
	"os"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/validatorerr"
	"github.com/bwmarrin/discordgo"
)

type DiscordGoSession interface {
	GuildMember(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error)
}

type ValidatorDiscord struct {
	session     DiscordGoSession
	adminRoleID string
	mentorRoleID string
}

func NewValidatorDiscord(s DiscordGoSession) *ValidatorDiscord {
	return &ValidatorDiscord{
		session:      s,
		adminRoleID:  os.Getenv("DISCORD_ADMIN_ROLE_ID"),
		mentorRoleID: os.Getenv("DISCORD_MENTOR_ROLE_ID"),
	}
}

func (validator *ValidatorDiscord) hasRole(guildID, userID, roleID string) (bool, error) {
	if roleID == "" {
		return false, validatorerr.NewInternalError("role ID is not configured", fmt.Errorf("empty role ID"))
	}
	
	member, err := validator.session.GuildMember(guildID, userID)
	if err != nil {
		return false, validatorerr.NewInternalError("could not get member", err)
	}
	
	for _, role := range member.Roles {
		if role == roleID {
			return true, nil
		}
	}

	return false, nil
}

func (validator *ValidatorDiscord) IsAdmin(guildID, userID string) (bool, error) {
	return validator.hasRole(guildID, userID, validator.adminRoleID)
}

func (validator *ValidatorDiscord) IsMentor(guildID, userID string) (bool, error) {
	return validator.hasRole(guildID, userID, validator.mentorRoleID)
}
