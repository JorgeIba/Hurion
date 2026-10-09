package mentor

import (
	"crypto/rand"
	"errors"

	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/providers"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/bwmarrin/discordgo"
)

type ACInteractionController struct {
	services         *services.Services
	validator        providers.DiscordValidator
	fetcher          providers.DiscordFetcher
	responder        dispatcher.InteractionResponder
	attachments      *jsonFileTransport
	validatedBatches *validatedACBatchCache
}

func NewACInteractionController(
	svcs *services.Services,
	validator providers.DiscordValidator,
	fetcher providers.DiscordFetcher,
	responder dispatcher.InteractionResponder,
) *ACInteractionController {
	return &ACInteractionController{
		services:         svcs,
		validator:        validator,
		fetcher:          fetcher,
		responder:        responder,
		attachments:      newJSONFileTransport(nil),
		validatedBatches: newValidatedACBatchCache(nil),
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

// HandleCreateACBatchModalSubmitInteraction validates without writing.
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
	if len(issues) > 0 {
		ctrl.editInteractionResponse(event.Interaction, batchParseReceipt(len(entries), issues))
		return
	}

	preparer := batchPreparer{batch: ctrl.services.Batch, fetcher: ctrl.fetcher}
	prepared, validationIssues, err := preparer.prepareACEntries(event.GuildID, entries)
	if err != nil {
		logger.Log.Errorf("Failed to resolve AC batch students: %v", err)
		ctrl.editInteractionResponse(event.Interaction, "Could not resolve a batch student from Discord. No changes were applied. Try again.")
		return
	}

	if len(validationIssues) > 0 {
		ctrl.editInteractionResponse(event.Interaction, acBatchValidationReceipt(entries, validationIssues))
		return
	}

	key := validatedBatchCacheKey{batchID: rand.Text(), guildID: event.GuildID, userID: event.Member.User.ID}
	available := ctrl.validatedBatches.saveValidatedBatch(key, prepared)
	if !available {
		ctrl.editInteractionResponse(event.Interaction, discord.BatchUnavailableMessage)
		return
	}

	preview := discord.BuildValidatedBatchPreview(
		discord.FormatACBatchValidationResult(prepared), discord.ACBatchCancelCustomID, key.batchID,
	)
	if _, err := ctrl.responder.EditResponse(event.Interaction, preview); err != nil {
		ctrl.validatedBatches.consumeBatch(key)
	}
}

func (ctrl *ACInteractionController) HandleCancelACBatchInteraction(event *discordgo.InteractionCreate) {
	batchID, err := discord.ParseBatchCancelID(event.MessageComponentData(), discord.ACBatchCancelCustomID)
	if err != nil {
		ctrl.responder.RespondPrivately(event.Interaction, err.Error())
		return
	}

	if event.Member == nil || event.Member.User == nil || event.Member.User.ID == "" {
		ctrl.responder.RespondPrivately(event.Interaction, "Batch commands require an identified server member.")
		return
	}

	key := validatedBatchCacheKey{batchID: batchID, guildID: event.GuildID, userID: event.Member.User.ID}
	if _, available := ctrl.validatedBatches.loadValidatedBatch(key); !available {
		ctrl.responder.RespondPrivately(event.Interaction, discord.BatchUnavailableMessage)
		return
	}

	if err := ctrl.responder.Respond(event.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate}); err != nil {
		return
	}

	if err := ctrl.authorizeACInteraction(event); err != nil {
		ctrl.editInteractionResponse(event.Interaction, err.Error())
		return
	}

	_, cancelled := ctrl.validatedBatches.consumeBatch(key)
	ctrl.responder.EditResponse(event.Interaction, discord.BuildBatchCancellationResult(cancelled))
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
