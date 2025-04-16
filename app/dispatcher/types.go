package dispatcher

import "github.com/bwmarrin/discordgo"

type MessageType int

const (
	SimpleMessage MessageType = iota
	ReplyMessage
	EmbedMessage
	Reaction
	FileMessage
)

type MessageResponse struct {
	Type      MessageType
	ChannelID string
	Content   string
	ReplyTo   string                  // optional
	Embed     *discordgo.MessageEmbed // optional
	Reactions []string                // optional
	Files     []*discordgo.File       // optional
}
