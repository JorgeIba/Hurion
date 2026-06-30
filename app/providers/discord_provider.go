package providers

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

func (p *discordProvider) BotUserID() string {
	state := p.session.State()
	if state != nil && state.User != nil {
		return state.User.ID
	}
	return ""
}
