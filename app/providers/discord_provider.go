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
