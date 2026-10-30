package dispatcher

import "github.com/bwmarrin/discordgo"

// InteractionResponder sends initial responses, edits or deletes their original result,
// publishes channel messages, and delivers private followups after acknowledgement.
type InteractionResponder interface {
	Respond(*discordgo.Interaction, *discordgo.InteractionResponse) error
	RespondPrivately(*discordgo.Interaction, string) error
	Publish(*discordgo.Interaction, *discordgo.MessageSend) (*discordgo.Message, error)
	FollowupPrivately(*discordgo.Interaction, string) error
	DeleteResponse(*discordgo.Interaction) error
	EditResponse(*discordgo.Interaction, *discordgo.WebhookEdit) (*discordgo.Message, error)
}

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
	Type                   MessageType
	ChannelID              string
	Content                string
	ReplyTo                string                       // optional
	Embed                  *discordgo.MessageEmbed      // optional
	Embeds                 []*discordgo.MessageEmbed    // optional
	Reactions              []string                     // optional
	ClearSpecificReactions []string                     // optional, specific emojis to clear before adding new ones
	Files                  []*discordgo.File            // optional
	Components             []discordgo.MessageComponent // optional
	ConfirmationArgs       []string                     // optional, canonical args for delayed confirmation execution

	MentionPolicy   MentionPolicy
	AllowedMentions *discordgo.MessageAllowedMentions // used when MentionPolicyCustom
}
