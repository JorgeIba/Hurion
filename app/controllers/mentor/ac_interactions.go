package mentor

import (
	"crypto/rand"
	"errors"

	controllers_common "github.com/JorgeIba/Hurion/app/controllers/common"
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/providers"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
)

type ACInteractionController struct {
	services         *services.Services
	validator        providers.DiscordValidator
	fetcher          providers.DiscordFetcher
	responder        dispatcher.InteractionResponder
	attachments      *jsonTransport
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
		attachments:      newJSONTransport(nil),
		validatedBatches: newValidatedACBatchCache(nil),
	}
}

// HandleAddACBatchInteraction opens the JSON intake form for explicit student ACs.
func (ctrl *ACInteractionController) HandleAddACBatchInteraction(event *discordgo.InteractionCreate) {
	if err := ctrl.authorizeACInteraction(event); err != nil {
		ctrl.responder.RespondPrivately(event.Interaction, err.Error())
		return
	}
	ctrl.responder.Respond(event.Interaction, interactiontransport.BuildACBatchModalResponse())
}

// HandleCreateACBatchModalSubmitInteraction validates without writing.
func (ctrl *ACInteractionController) HandleCreateACBatchModalSubmitInteraction(event *discordgo.InteractionCreate) {
	// Acknowledge the submission and authorize the actor.
	if err := ctrl.deferACInteractionResponse(event.Interaction); err != nil {
		return
	}

	if err := ctrl.authorizeACInteraction(event); err != nil {
		ctrl.editInteractionResponse(event.Interaction, err.Error())
		return
	}

	// Read the form and JSON document, then decode the batch input.
	source, err := interactiontransport.ParseACBatchModalSource(event.ModalSubmitData())
	if err != nil {
		ctrl.editInteractionResponse(event.Interaction, "Could not read the batch form: "+err.Error())
		return
	}

	input, err := ctrl.attachments.read(source)
	if err != nil {
		ctrl.editInteractionResponse(event.Interaction, discord.FormatJSONReadError(mapJSONReadIssue(err)))
		return
	}

	entries, issues := parseACBatchInput(input)
	if len(issues) > 0 {
		ctrl.editInteractionResponse(event.Interaction, discord.FormatJSONParseResult(len(entries), mapBatchParseIssues(issues)))
		return
	}

	// Validate input values before external lookups.
	validationIssues := entries.validate(validateACInput)
	if len(validationIssues) > 0 {
		ctrl.editInteractionResponse(event.Interaction, discord.FormatBatchValidationErrors(mapACBatchValidationIssues(entries, validationIssues)))
		return
	}

	// Adapt the input and resolve Discord principals.
	inputs := mapACBatchInputToCreateACParams(entries)
	userIDs := make([]string, len(entries))
	for index, entry := range entries {
		userIDs[index] = service_model.NormalizeACUserID(entry.UserID)
	}
	students, err := controllers_common.GetMemberPrincipals(ctrl.fetcher, event.GuildID, userIDs)

	if err != nil {
		logger.Log.Errorf("Failed to resolve AC batch students: %v", err)
		ctrl.editInteractionResponse(event.Interaction, "Could not resolve a batch student from Discord. No changes were applied. Try again.")
		return
	}

	// Validate the complete batch with the service.
	prepared, domainIssues := ctrl.services.Batch.PrepareACBatch(inputs, students)
	if len(domainIssues) > 0 {
		ctrl.editInteractionResponse(event.Interaction, discord.FormatBatchValidationErrors(mapACBatchValidationIssues(entries, mapBatchEntryErrors(domainIssues))))
		return
	}

	// Save the validated snapshot and show its preview.
	key := validatedBatchCacheKey{
		batchID: rand.Text(),
		guildID: event.GuildID,
		userID:  event.Member.User.ID,
	}
	available := ctrl.validatedBatches.saveValidatedBatch(key, prepared)
	if !available {
		ctrl.editInteractionResponse(event.Interaction, discord.BatchUnavailableMessage)
		return
	}

	preview := discord.BuildValidatedBatchPreview(
		discord.FormatACBatchValidationResult(prepared), interactiontransport.ACBatchCancelID(key.batchID),
	)
	if _, err := ctrl.responder.EditResponse(event.Interaction, preview); err != nil {
		ctrl.validatedBatches.consumeBatch(key)
	}
}

func (ctrl *ACInteractionController) HandleCancelACBatchInteraction(event *discordgo.InteractionCreate) {
	if event.MessageComponentData().ComponentType != discordgo.ButtonComponent {
		ctrl.responder.RespondPrivately(event.Interaction, "invalid batch cancellation button")
		return
	}

	batchID, err := interactiontransport.ParseACBatchCancelID(event.MessageComponentData().CustomID)
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
		ctrl.responder.FollowupPrivately(event.Interaction, err.Error())
		return
	}

	_, cancelled := ctrl.validatedBatches.consumeBatch(key)
	if !cancelled {
		return
	}
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
