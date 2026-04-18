package providers

import (
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/discorderr"
)

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
