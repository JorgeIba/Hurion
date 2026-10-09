package mentor

import (
	"errors"

	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/providers"
	"github.com/bwmarrin/discordgo"
)

type ACInteractionController struct {
	validator   providers.DiscordValidator
	responder   dispatcher.InteractionResponder
	attachments *jsonFileTransport
}

func NewACInteractionController(validator providers.DiscordValidator, responder dispatcher.InteractionResponder) *ACInteractionController {
	return &ACInteractionController{
		validator: validator, responder: responder,
		attachments: newJSONFileTransport(nil),
	}
}

// HandleAddACBatchInteraction opens the JSON intake form for explicit student ACs.
func (ctrl *ACInteractionController) HandleAddACBatchInteraction(event *discordgo.InteractionCreate) {
	if err := ctrl.authorizeACInteraction(event); err != nil {
		ctrl.responder.RespondPrivately(event.Interaction, err.Error())
		return
	}
	ctrl.responder.Respond(event.Interaction, discord.BuildACBatchModalResponse())
}

// HandleCreateACBatchModalSubmitInteraction parses without preparing or writing.
func (ctrl *ACInteractionController) HandleCreateACBatchModalSubmitInteraction(event *discordgo.InteractionCreate) {
	if err := ctrl.deferACInteractionResponse(event.Interaction); err != nil {
		return
	}
	if err := ctrl.authorizeACInteraction(event); err != nil {
		ctrl.editInteractionResponse(event.Interaction, err.Error())
		return
	}
	source, err := discord.ParseACBatchModalSource(event.ModalSubmitData())
	if err != nil {
		ctrl.editInteractionResponse(event.Interaction, "Could not read the batch form: "+err.Error())
		return
	}
	input, err := ctrl.attachments.read(source)
	if err != nil {
		ctrl.editInteractionResponse(event.Interaction, "Could not read the batch form: "+err.Error())
		return
	}
	entries, issues := parseACBatch(input)
	ctrl.editInteractionResponse(event.Interaction, batchParseReceipt(len(entries), issues))
}

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

func (ctrl *ACInteractionController) authorizeACInteraction(event *discordgo.InteractionCreate) error {
	if event.Member == nil || event.Member.User == nil || event.Member.User.ID == "" {
		return errors.New("AC commands require an identified server member.")
	}
	allowed, err := canManageACs(ctrl.validator, event.GuildID, event.Member.User.ID)
	if err != nil {
		logger.Log.Errorf("Failed to verify interaction AC-management permissions: %v", err)
		return errors.New(messageErrVerifying)
	}
	if !allowed {
		return errors.New(messageErrACPermissions)
	}
	return nil
}
