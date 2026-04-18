package providers

import (
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/discorderr"
	"github.com/bwmarrin/discordgo"
)

func (provider *discordProvider) FetchMessage(channelID, messageID string) (*discordgo.Message, error) {
	if provider.session == nil {
		return nil, discorderr.NewInternalError("discord session not configured", fmt.Errorf("nil session"))
	}

	message, err := provider.session.ChannelMessage(channelID, messageID)
	if err != nil {
		return nil, discorderr.NewInternalError("could not fetch message", err)
	}

	return message, nil
}

func (provider *discordProvider) EditEmbed(channelID, messageID string, embed *discordgo.MessageEmbed) error {
	if provider.session == nil {
		return discorderr.NewInternalError("discord session not configured", fmt.Errorf("nil session"))
	}

	_, err := provider.session.ChannelMessageEditEmbed(channelID, messageID, embed)
	if err != nil {
		return discorderr.NewInternalError("could not edit embed", err)
	}

	return nil
}

func (provider *discordProvider) RemoveBotReaction(channelID, messageID, emojiID string) error {
	if provider.session == nil {
		return discorderr.NewInternalError("discord session not configured", fmt.Errorf("nil session"))
	}

	state := provider.session.State()
	if state == nil || state.User == nil {
		return discorderr.NewInternalError("discord session state is not available", fmt.Errorf("state or bot user is nil"))
	}

	err := provider.session.MessageReactionRemove(channelID, messageID, emojiID, state.User.ID)
	if err != nil {
		return discorderr.NewInternalError("could not remove reaction", err)
	}

	return nil
}

func (provider *discordProvider) GetCachedMember(guildID, userID string) (*discordgo.Member, error) {
	if provider.session == nil {
		return nil, discorderr.NewInternalError("discord session not configured", fmt.Errorf("nil session"))
	}

	state := provider.session.State()
	if state == nil {
		return nil, discorderr.NewInternalError("discord session state is not available", fmt.Errorf("nil state"))
	}

	member, err := state.Member(guildID, userID)
	if err != nil {
		return nil, discorderr.NewInternalError("could not get cached member", err)
	}

	return member, nil
}
