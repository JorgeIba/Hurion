package dispatcher

import (
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/bwmarrin/discordgo"
)

type discordInteractionClient interface {
	ChannelMessageSendComplex(string, *discordgo.MessageSend, ...discordgo.RequestOption) (*discordgo.Message, error)
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
		logger.Log.Errorf("Failed to respond to interaction: interaction=%s error=%v", interaction.ID, err)
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
		logger.Log.Errorf("Failed to delete interaction response: interaction=%s error=%v", interaction.ID, err)
	}

	return err
}

func (r *DiscordInteractionResponder) EditResponse(interaction *discordgo.Interaction, response *discordgo.WebhookEdit) (*discordgo.Message, error) {
	message, err := r.client.InteractionResponseEdit(interaction, response)
	if err != nil {
		logger.Log.Errorf("Failed to edit interaction response: interaction=%s error=%v", interaction.ID, err)
	}
	return message, err
}

// Publish sends a standalone channel message without interaction response metadata.
func (r *DiscordInteractionResponder) Publish(interaction *discordgo.Interaction, response *discordgo.MessageSend) (*discordgo.Message, error) {
	message, err := r.client.ChannelMessageSendComplex(interaction.ChannelID, response)
	if err != nil {
		logger.Log.Errorf("Failed to publish channel message: interaction=%s error=%v", interaction.ID, err)
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
		logger.Log.Errorf("Failed to send private interaction followup: interaction=%s error=%v", interaction.ID, err)
	}
	return err
}
