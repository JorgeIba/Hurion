package dispatcher

import "github.com/bwmarrin/discordgo"

type MessageType int

const (
	SimpleMessage MessageType = iota
	ReplyMessage
	EmbedMessage
	Reaction
	FileMessage
	PaginationEmbed
)

type MessageResponse struct {
	Type      MessageType
	Session   *discordgo.Session
	ChannelID string
	Content   string
	ReplyTo   string                    // optional
	Embed     *discordgo.MessageEmbed   // optional
	Embeds    []*discordgo.MessageEmbed // optional
	Reactions []string                  // optional
	Files     []*discordgo.File         // optional
}
