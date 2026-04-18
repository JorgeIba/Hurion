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
	ConfirmationMessage
)

type MentionPolicy int

const (
	// MentionPolicyNone suppresses all mentions (safe default).
	MentionPolicyNone MentionPolicy = iota
	// MentionPolicyUsers allows only user mentions.
	MentionPolicyUsers
	// MentionPolicyAll allows users, roles, and everyone mentions.
	MentionPolicyAll
	// MentionPolicyCustom uses MessageResponse.AllowedMentions as-is.
	MentionPolicyCustom
)

type MessageResponse struct {
	Type      MessageType
	ChannelID string
	Content   string
	ReplyTo   string                    // optional
	Embed     *discordgo.MessageEmbed   // optional
	Embeds    []*discordgo.MessageEmbed // optional
	Reactions []string                  // optional
	Files     []*discordgo.File         // optional

	MentionPolicy   MentionPolicy
	AllowedMentions *discordgo.MessageAllowedMentions // used when MentionPolicyCustom
}
