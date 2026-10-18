package mentor

import (
	"crypto/rand"
	"errors"

	controllers_common "github.com/JorgeIba/Hurion/app/controllers/common"
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/providers"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
)

type ACInteractionController struct {
	services             *services.Services
	validator            providers.DiscordValidator
	fetcher              providers.DiscordFetcher
	responder            dispatcher.InteractionResponder
	attachments          *jsonTransport
	attachmentDownloader *interactiontransport.AttachmentDownloader
	validatedBatches     *validatedACBatchCache
}

func NewACInteractionController(
	svcs *services.Services,
	validator providers.DiscordValidator,
	fetcher providers.DiscordFetcher,
	responder dispatcher.InteractionResponder,
) *ACInteractionController {
	downloader := interactiontransport.NewAttachmentDownloader(nil)
	return &ACInteractionController{
		services:             svcs,
		validator:            validator,
		fetcher:              fetcher,
		responder:            responder,
		attachments:          &jsonTransport{downloader: downloader},
		attachmentDownloader: downloader,
		validatedBatches:     newValidatedACBatchCache(nil),
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
		ctrl.editInteractionResponse(event.Interaction, discord.FormatBatchValidationErrors(mapACBatchValidationIssues(entries, validationIssues, nil)))
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
	prepared, domainIssues, err := ctrl.services.Solution.PrepareCreationBatch(inputs, students)
	if err != nil {
		logger.Log.Errorf("Failed to validate AC batch: %v", err)
		ctrl.editInteractionResponse(event.Interaction, "Could not validate the AC batch. No changes were applied. Try again.")
		return
	}
	if len(domainIssues) > 0 {
		for _, issue := range domainIssues {
			logger.Log.Warnf("AC batch validation rejected: guild=%s actor=%s entry=%d error=%v", event.GuildID, event.Member.User.ID, issue.Entry, issue.Err)
		}
		ctrl.editInteractionResponse(event.Interaction, discord.FormatBatchValidationErrors(mapACBatchValidationIssues(entries, mapBatchEntryErrors(domainIssues), students)))
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
		discord.FormatACBatchValidationResult(prepared),
		interactiontransport.ACBatchConfirmID(key.batchID),
		interactiontransport.ACBatchCancelID(key.batchID),
	)
	if _, err := ctrl.responder.EditResponse(event.Interaction, preview); err != nil {
		ctrl.validatedBatches.consumeBatch(key)
	}
}

// HandleConfirmACBatchInteraction claims a validated batch before executing it.
func (ctrl *ACInteractionController) HandleConfirmACBatchInteraction(event *discordgo.InteractionCreate) {
	// Read the button context and identify the submitting member.
	if event.MessageComponentData().ComponentType != discordgo.ButtonComponent {
		ctrl.responder.RespondPrivately(event.Interaction, "invalid batch action button")
		return
	}

	batchID, err := interactiontransport.ParseACBatchConfirmID(event.MessageComponentData().CustomID)
	if err != nil {
		ctrl.responder.RespondPrivately(event.Interaction, err.Error())
		return
	}
	if event.Member == nil || event.Member.User == nil || event.Member.User.ID == "" {
		ctrl.responder.RespondPrivately(event.Interaction, "Batch commands require an identified server member.")
		return
	}

	// Find an unexpired batch owned by this member and guild.
	key := validatedBatchCacheKey{batchID: batchID, guildID: event.GuildID, userID: event.Member.User.ID}
	snapshot, available := ctrl.validatedBatches.loadValidatedBatch(key)
	if !available {
		ctrl.responder.RespondPrivately(event.Interaction, discord.BatchUnavailableMessage)
		return
	}

	// Acknowledge the click and verify current authority.
	if err := ctrl.responder.Respond(event.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate}); err != nil {
		return
	}
	if err := ctrl.authorizeACInteraction(event); err != nil {
		ctrl.responder.FollowupPrivately(event.Interaction, err.Error())
		return
	}

	// Resolve current student identities and roles before claiming the batch.
	userIDs := make([]string, len(snapshot))
	for index, entry := range snapshot {
		userIDs[index] = entry.User.UserID
	}
	students, err := controllers_common.GetMemberPrincipals(ctrl.fetcher, event.GuildID, userIDs)
	if err != nil {
		logger.Log.Errorf("Failed to resolve AC batch students before confirmation: %v", err)
		ctrl.responder.FollowupPrivately(event.Interaction, "Could not resolve a batch student from Discord. Try confirming again.")
		return
	}

	// Claim once, then revalidate and execute through the batch service.
	batch, claimed := ctrl.validatedBatches.consumeBatch(key)
	if !claimed {
		return
	}
	result, issues, err := ctrl.services.Solution.ExecuteCreationBatch(batch, students)

	// Present validation failures or the acknowledged execution outcome.
	var response *discordgo.WebhookEdit
	if _, preparationFailed := errors.AsType[*serviceerr.BatchPreparationError](err); preparationFailed {
		logger.Log.Errorf("Failed to revalidate AC batch %s: %v", batchID, err)
		response = discord.BuildBatchValidationFailure("Could not validate the AC batch. No changes were applied. Submit the batch again.")
	} else if len(issues) > 0 {
		for _, issue := range issues {
			logger.Log.Warnf("AC batch %s confirmation rejected: guild=%s actor=%s entry=%d error=%v", batchID, event.GuildID, event.Member.User.ID, issue.Entry, issue.Err)
		}
		locations := mapACBatchExecutionIssues(batch, issues, students)
		response = discord.BuildBatchValidationFailure(discord.FormatBatchValidationErrors(locations))
	} else {
		if err != nil {
			logger.Log.Errorf("Failed to execute AC batch %s: %v", batchID, err)
		}
		response = discord.BuildACBatchExecutionResult(result, err != nil)
	}
	// Log and deliver the result without retrying consumed operations.
	logger.Log.Infof("AC batch %s outcome: guild=%s actor=%s entries=%d result=%v validationIssues=%d error=%v", batchID, event.GuildID, event.Member.User.ID, len(batch), result, len(issues), err)
	if _, err := ctrl.responder.EditResponse(event.Interaction, response); err != nil {
		logger.Log.Errorf("Failed to deliver AC batch %s result: %v", batchID, err)
	}
}

func (ctrl *ACInteractionController) HandleCancelACBatchInteraction(event *discordgo.InteractionCreate) {
	// Read the button context and identify the submitting member.
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

	// Find an unexpired batch owned by this member and guild.
	key := validatedBatchCacheKey{batchID: batchID, guildID: event.GuildID, userID: event.Member.User.ID}
	if _, available := ctrl.validatedBatches.loadValidatedBatch(key); !available {
		ctrl.responder.RespondPrivately(event.Interaction, discord.BatchUnavailableMessage)
		return
	}

	// Acknowledge the click and verify current authority.
	if err := ctrl.responder.Respond(event.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate}); err != nil {
		return
	}

	if err := ctrl.authorizeACInteraction(event); err != nil {
		ctrl.responder.FollowupPrivately(event.Interaction, err.Error())
		return
	}

	// Consume the pending batch and remove its preview actions.
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
