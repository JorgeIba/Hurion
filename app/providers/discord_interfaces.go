package providers

import "github.com/bwmarrin/discordgo"

// DiscordGoSession abstracts the discordgo session dependency used by providers.
//
//go:generate go tool mockgen -destination=mock_providers/mock_session.go -package=mock_providers . DiscordGoSession
type DiscordGoSession interface {
	GuildMember(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error)
	ChannelMessage(channelID, messageID string, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessageEditEmbed(channelID, messageID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error)
	MessageReactionRemove(channelID, messageID, emojiID, userID string, options ...discordgo.RequestOption) error
	State() *discordgo.State
}

// DiscordValidator focuses on permission checks.
//
//go:generate go tool mockgen -destination=mock_providers/mock_validator.go -package=mock_providers . DiscordValidator
type DiscordValidator interface {
	IsAdmin(guildID, userID string) (bool, error)
	IsMentor(guildID, userID string) (bool, error)
}

// DiscordFetcher focuses on fetching Discord data.
//
//go:generate go tool mockgen -destination=mock_providers/mock_fetcher.go -package=mock_providers . DiscordFetcher
type DiscordFetcher interface {
	FetchMessage(channelID, messageID string) (*discordgo.Message, error)
	GetCachedMember(guildID, userID string) (*discordgo.Member, error)
}

// DiscordMessageModifier focuses on Discord message mutation operations.
//
//go:generate go tool mockgen -destination=mock_providers/mock_modifier.go -package=mock_providers . DiscordMessageModifier
type DiscordMessageModifier interface {
	EditEmbed(channelID, messageID string, embed *discordgo.MessageEmbed) error
	RemoveBotReaction(channelID, messageID, emojiID string) error
}
type DiscordProvider interface {
	DiscordValidator
	DiscordFetcher
	DiscordMessageModifier
}
