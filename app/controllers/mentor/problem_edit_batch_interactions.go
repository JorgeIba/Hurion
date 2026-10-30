package mentor

import (
	"crypto/rand"
	"errors"

	"github.com/JorgeIba/Hurion/app/controllers/adapters"
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/bwmarrin/discordgo"
)

func (ctrl *ProblemInteractionController) HandleEditProblemBatchInteraction(event *discordgo.InteractionCreate) {
	if err := ctrl.validateProblemActor(event); err != nil {
		ctrl.responder.RespondPrivately(event.Interaction, err.Error())
		return
	}

	ctrl.responder.Respond(event.Interaction, interactiontransport.BuildProblemEditBatchModalResponse())
}

// HandleEditProblemBatchModalSubmitInteraction validates without writing.
func (ctrl *ProblemInteractionController) HandleEditProblemBatchModalSubmitInteraction(event *discordgo.InteractionCreate) {
	// Acknowledge the submission and validate the actor identity.
	if err := ctrl.deferProblemInteractionResponse(event.Interaction); err != nil {
		return
	}

	if err := ctrl.validateProblemActor(event); err != nil {
		ctrl.editInteractionResponse(event.Interaction, err.Error())
		return
	}

	// Read the form and JSON document, then decode the batch input.
	source, err := interactiontransport.ParseProblemEditBatchModalSource(event.ModalSubmitData())
	if err != nil {
		ctrl.editInteractionResponse(event.Interaction, "Could not read the batch form: "+err.Error())
		return
	}

	input, err := ctrl.attachments.read(source)
	if err != nil {
		ctrl.editInteractionResponse(event.Interaction, discord.FormatJSONReadError(mapJSONReadIssue(err)))
		return
	}

	entries, issues := parseProblemEditBatchInput(input)
	if len(issues) > 0 {
		ctrl.editInteractionResponse(event.Interaction, discord.FormatJSONParseResult(len(entries), mapBatchParseIssues(issues)))
		return
	}

	// Validate input values before external lookups.
	validationIssues := entries.validate(validateProblemEditInput)
	if len(validationIssues) > 0 {
		ctrl.editInteractionResponse(event.Interaction, discord.FormatBatchValidationErrors(mapProblemEditBatchValidationIssues(entries, validationIssues)))
		return
	}

	// Adapt the input and validate the complete batch with the service.
	inputs := mapProblemEditBatchInputToPatchProblemParams(entries)
	prepared, domainIssues, err := ctrl.services.Problem.PrepareProblemPatchBatch(services.UserActor(event.Member.User.ID), inputs)
	if err != nil {
		logger.Log.Errorf("Failed to validate problem edit batch: %v", err)
		if _, denied := errors.AsType[*serviceerr.PermissionDeniedError](err); denied {
			ctrl.editInteractionResponse(event.Interaction, adapters.MapError(err, adapters.ActionMentorUpdateProblem))
		} else {
			ctrl.editInteractionResponse(event.Interaction, "Could not validate the problem edit batch. No changes were applied. Try again.")
		}
		return
	}

	if len(domainIssues) > 0 {
		for _, issue := range domainIssues {
			logger.Log.Warnf("Problem edit batch validation rejected: guild=%s actor=%s entry=%d error=%v", event.GuildID, event.Member.User.ID, issue.Entry, issue.Err)
		}
		ctrl.editInteractionResponse(event.Interaction, discord.FormatBatchValidationErrors(mapProblemEditBatchValidationIssues(entries, mapBatchEntryErrors(domainIssues))))
		return
	}

	// Save the validated snapshot and show its preview.
	key := validatedBatchCacheKey{
		batchID: rand.Text(),
		guildID: event.GuildID,
		userID:  event.Member.User.ID,
	}
	available := ctrl.validatedEditBatches.saveValidatedBatch(key, prepared)
	if !available {
		ctrl.editInteractionResponse(event.Interaction, discord.BatchUnavailableMessage)
		return
	}

	preview := discord.BuildValidatedBatchPreview(
		discord.FormatProblemEditBatchValidationResult(prepared),
		interactiontransport.ProblemEditBatchConfirmID(key.batchID),
		interactiontransport.ProblemEditBatchCancelID(key.batchID),
	)
	if _, err := ctrl.responder.EditResponse(event.Interaction, preview); err != nil {
		ctrl.validatedEditBatches.consumeBatch(key)
	}
}

// HandleConfirmProblemEditBatchInteraction claims a validated batch before executing it.
func (ctrl *ProblemInteractionController) HandleConfirmProblemEditBatchInteraction(event *discordgo.InteractionCreate) {
	// Read the button context and identify the submitting member.
	if event.MessageComponentData().ComponentType != discordgo.ButtonComponent {
		ctrl.responder.RespondPrivately(event.Interaction, "invalid batch action button")
		return
	}

	batchID, err := interactiontransport.ParseProblemEditBatchConfirmID(event.MessageComponentData().CustomID)
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
	_, available := ctrl.validatedEditBatches.loadValidatedBatch(key)
	if !available {
		ctrl.responder.RespondPrivately(event.Interaction, discord.BatchUnavailableMessage)
		return
	}

	// Acknowledge the click and validate the actor identity.
	if err := ctrl.responder.Respond(event.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate}); err != nil {
		return
	}

	if err := ctrl.validateProblemActor(event); err != nil {
		ctrl.responder.FollowupPrivately(event.Interaction, err.Error())
		return
	}

	// Claim once, then revalidate and execute through the problem service.
	batch, claimed := ctrl.validatedEditBatches.consumeBatch(key)
	if !claimed {
		return
	}
	result, issues, err := ctrl.services.Problem.ExecuteProblemPatchBatch(services.UserActor(event.Member.User.ID), batch)

	// Present validation failures or the acknowledged execution outcome.
	var response *discordgo.WebhookEdit
	if _, denied := errors.AsType[*serviceerr.PermissionDeniedError](err); denied {
		response = discord.BuildBatchValidationFailure(adapters.MapError(err, adapters.ActionMentorUpdateProblem))
	} else if _, preparationFailed := errors.AsType[*serviceerr.BatchPreparationError](err); preparationFailed {
		logger.Log.Errorf("Failed to revalidate problem edit batch %s: %v", batchID, err)
		response = discord.BuildBatchValidationFailure("Could not validate the problem edit batch. No changes were applied. Submit the batch again.")
	} else if len(issues) > 0 {
		locations := mapProblemEditBatchExecutionIssues(batch, issues)
		response = discord.BuildBatchValidationFailure(discord.FormatBatchValidationErrors(locations))
	} else {
		if err != nil {
			logger.Log.Errorf("Failed to execute Problem edit batch %s: %v", batchID, err)
		}
		response = discord.BuildProblemEditBatchExecutionResult(result, err != nil)
	}

	// Log and deliver the result without retrying consumed operations.
	logger.Log.Infof("Problem edit batch %s outcome: guild=%s actor=%s entries=%d result=%v validationIssues=%d error=%v", batchID, event.GuildID, event.Member.User.ID, len(batch), result.PatchedProblemIDs, len(issues), err)
	if _, err := ctrl.responder.EditResponse(event.Interaction, response); err != nil {
		logger.Log.Errorf("Failed to deliver Problem edit batch %s result: %v", batchID, err)
	}
}

func (ctrl *ProblemInteractionController) HandleCancelProblemEditBatchInteraction(event *discordgo.InteractionCreate) {
	// Read the button context and identify the submitting member.
	if event.MessageComponentData().ComponentType != discordgo.ButtonComponent {
		ctrl.responder.RespondPrivately(event.Interaction, "invalid batch cancellation button")
		return
	}

	batchID, err := interactiontransport.ParseProblemEditBatchCancelID(event.MessageComponentData().CustomID)
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
	// A forged or stale button must not update another actor's preview.
	_, available := ctrl.validatedEditBatches.loadValidatedBatch(key)
	if !available {
		ctrl.responder.RespondPrivately(event.Interaction, discord.BatchUnavailableMessage)
		return
	}

	// Acknowledge the click and validate the actor identity.
	if err := ctrl.responder.Respond(event.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate}); err != nil {
		return
	}

	if err := ctrl.validateProblemActor(event); err != nil {
		ctrl.responder.FollowupPrivately(event.Interaction, err.Error())
		return
	}

	// Consume the pending batch and remove its preview actions.
	_, cancelled := ctrl.validatedEditBatches.consumeBatch(key)
	if !cancelled {
		return
	}
	ctrl.responder.EditResponse(event.Interaction, discord.BuildBatchCancellationResult(cancelled))
}
