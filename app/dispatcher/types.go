package dispatcher

import "github.com/bwmarrin/discordgo"

type MessageType int

const (
	SimpleMessage MessageType = iota
	ReplyMessage
	EmbedMessage
	Reaction
)

type MessageResponse struct {
	Type      MessageType
	ChannelID string
	Content   string
	ReplyTo   string                  // optional
	Embed     *discordgo.MessageEmbed // optional
	Reactions []string                // optional
}
