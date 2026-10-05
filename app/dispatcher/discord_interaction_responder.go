package dispatcher

import (
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/bwmarrin/discordgo"
)

type discordInteractionClient interface {
	InteractionRespond(*discordgo.Interaction, *discordgo.InteractionResponse, ...discordgo.RequestOption) error
	InteractionResponseEdit(*discordgo.Interaction, *discordgo.WebhookEdit, ...discordgo.RequestOption) (*discordgo.Message, error)
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

func (r *DiscordInteractionResponder) EditResponse(interaction *discordgo.Interaction, response *discordgo.WebhookEdit) (*discordgo.Message, error) {
	message, err := r.client.InteractionResponseEdit(interaction, response)
	if err != nil {
		logger.Log.Errorf("Failed to edit interaction response: %v", err)
	}
	return message, err
}
