package providers

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type discordgoSessionAdapter struct {
	session *discordgo.Session
}

func NewDiscordGoSessionAdapter(session *discordgo.Session) DiscordGoSession {
	return &discordgoSessionAdapter{session: session}
}

func (adapter *discordgoSessionAdapter) GuildMember(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error) {
	if adapter == nil || adapter.session == nil {
		return nil, fmt.Errorf("session is nil")
	}

	return adapter.session.GuildMember(guildID, userID, options...)
}

func (adapter *discordgoSessionAdapter) ChannelMessage(channelID, messageID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	if adapter == nil || adapter.session == nil {
		return nil, fmt.Errorf("session is nil")
	}

	return adapter.session.ChannelMessage(channelID, messageID, options...)
}

func (adapter *discordgoSessionAdapter) ChannelMessageEditEmbed(channelID, messageID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	if adapter == nil || adapter.session == nil {
		return nil, fmt.Errorf("session is nil")
	}

	return adapter.session.ChannelMessageEditEmbed(channelID, messageID, embed, options...)
}

func (adapter *discordgoSessionAdapter) MessageReactionRemove(channelID, messageID, emojiID, userID string, options ...discordgo.RequestOption) error {
	if adapter == nil || adapter.session == nil {
		return fmt.Errorf("session is nil")
	}

	return adapter.session.MessageReactionRemove(channelID, messageID, emojiID, userID, options...)
}

func (adapter *discordgoSessionAdapter) State() *discordgo.State {
	if adapter == nil || adapter.session == nil {
		return nil
	}

	return adapter.session.State
}
