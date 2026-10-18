package dispatcher

import (
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/bwmarrin/discordgo"
)

type discordInteractionClient interface {
	InteractionRespond(*discordgo.Interaction, *discordgo.InteractionResponse, ...discordgo.RequestOption) error
	InteractionResponseDelete(*discordgo.Interaction, ...discordgo.RequestOption) error
	InteractionResponseEdit(*discordgo.Interaction, *discordgo.WebhookEdit, ...discordgo.RequestOption) (*discordgo.Message, error)
	FollowupMessageCreate(*discordgo.Interaction, bool, *discordgo.WebhookParams, ...discordgo.RequestOption) (*discordgo.Message, error)
}

// DiscordInteractionResponder delivers responses synchronously to avoid waiting behind
// channel messages during Discord's three-second initial-response deadline.
type DiscordInteractionResponder struct {
	client discordInteractionClient
}

func NewDiscordInteractionResponder(client discordInteractionClient) *DiscordInteractionResponder {
	return &DiscordInteractionResponder{client: client}
}

func (r *DiscordInteractionResponder) Respond(interaction *discordgo.Interaction, response *discordgo.InteractionResponse) error {
	err := r.client.InteractionRespond(interaction, response)
	if err != nil {
		logger.Log.Errorf("Failed to respond to interaction: %v", err)
	}
	return err
}

func (r *DiscordInteractionResponder) RespondPrivately(interaction *discordgo.Interaction, content string) error {
	return r.Respond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:         content,
			Flags:           discordgo.MessageFlagsEphemeral,
			AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}},
		},
	})
}

func (r *DiscordInteractionResponder) DeleteResponse(interaction *discordgo.Interaction) error {
	err := r.client.InteractionResponseDelete(interaction)
	if err != nil {
		logger.Log.Errorf("Failed to delete interaction response: %v", err)
	}

	return err
}

func (r *DiscordInteractionResponder) EditResponse(interaction *discordgo.Interaction, response *discordgo.WebhookEdit) (*discordgo.Message, error) {
	message, err := r.client.InteractionResponseEdit(interaction, response)
	if err != nil {
		logger.Log.Errorf("Failed to edit interaction response: %v", err)
	}
	return message, err
}

// Followup sends a public interaction followup. It is useful when a command
// should acknowledge privately but publish one shared result in the channel.
func (r *DiscordInteractionResponder) Followup(interaction *discordgo.Interaction, response *discordgo.WebhookParams) (*discordgo.Message, error) {
	message, err := r.client.FollowupMessageCreate(interaction, true, response)
	if err != nil {
		logger.Log.Errorf("Failed to send interaction followup: %v", err)
	}

	return message, err
}

// FollowupPrivately sends a private message after acknowledgement without editing the
// shared original message of a component interaction.
func (r *DiscordInteractionResponder) FollowupPrivately(interaction *discordgo.Interaction, content string) error {
	_, err := r.client.FollowupMessageCreate(interaction, true, &discordgo.WebhookParams{
		Content:         content,
		Flags:           discordgo.MessageFlagsEphemeral,
		AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}},
	})
	if err != nil {
		logger.Log.Errorf("Failed to send private interaction followup: %v", err)
	}
	return err
}
