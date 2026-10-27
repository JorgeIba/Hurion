package providers

type discordProvider struct{ session DiscordGoSession }

func NewProviderDiscord(s DiscordGoSession) DiscordProvider { return &discordProvider{session: s} }

func (p *discordProvider) BotUserID() string {
	state := p.session.State()
	if state != nil && state.User != nil {
		return state.User.ID
	}
	return ""
}
