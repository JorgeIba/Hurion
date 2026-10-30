package mentor

import (
	"errors"

	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/providers"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/bwmarrin/discordgo"
)

type ACInteractionController struct {
	services             *services.Services
	fetcher              providers.DiscordFetcher
	responder            dispatcher.InteractionResponder
	attachments          *jsonTransport
	attachmentDownloader *interactiontransport.AttachmentDownloader
	validatedBatches     *validatedACBatchCache
}

func NewACInteractionController(
	svcs *services.Services,
	fetcher providers.DiscordFetcher,
	responder dispatcher.InteractionResponder,
) *ACInteractionController {
	downloader := interactiontransport.NewAttachmentDownloader(nil)
	return &ACInteractionController{
		services:             svcs,
		fetcher:              fetcher,
		responder:            responder,
		attachments:          &jsonTransport{downloader: downloader},
		attachmentDownloader: downloader,
		validatedBatches:     newValidatedACBatchCache(nil),
	}
}

// deferACInteractionResponse acknowledges the interaction privately before slow work
// such as permission lookups or database operations, preventing Discord from timing
// out the command. The caller sends the final result with editInteractionResponse.
func (ctrl *ACInteractionController) deferACInteractionResponse(interaction *discordgo.Interaction) error {
	return ctrl.responder.Respond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	})
}

func (ctrl *ACInteractionController) editInteractionResponse(interaction *discordgo.Interaction, content string) {
	ctrl.responder.EditResponse(interaction, &discordgo.WebhookEdit{
		Content: &content, AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}},
	})
}

func (ctrl *ACInteractionController) validateACActor(event *discordgo.InteractionCreate) error {
	if event.Member == nil || event.Member.User == nil || event.Member.User.ID == "" {
		return errors.New("AC commands require an identified server member.")
	}
	return nil
}
