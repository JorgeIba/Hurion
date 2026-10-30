package mentor

import (
	"crypto/rand"
	"fmt"

	"github.com/JorgeIba/Hurion/app/controllers/adapters"
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/bwmarrin/discordgo"
)

const problemRemovalUnavailable = "This removal confirmation is no longer available. Run /problem remove again."

func (ctrl *ProblemInteractionController) HandleRemoveProblemInteraction(event *discordgo.InteractionCreate) {
	if !ctrl.validateProblemRemovalEvent(event, discordgo.InteractionApplicationCommand) {
		return
	}

	problemID, err := interactiontransport.ParseProblemRemoveOptions(event.ApplicationCommandData())
	if err != nil {
		ctrl.responder.RespondPrivately(event.Interaction, err.Error())
		return
	}

	if err := ctrl.deferProblemInteractionResponse(event.Interaction); err != nil {
		return
	}

	preview, err := ctrl.services.Problem.PrepareProblemRemoval(services.UserActor(event.Member.User.ID), problemID)
	if err != nil {
		ctrl.editInteractionResponse(event.Interaction, adapters.MapError(err, adapters.ActionMentorRemoveProblem, problemID))
		return
	}

	key := validatedBatchCacheKey{
		guildID: event.GuildID,
		userID:  event.Member.User.ID,
		batchID: rand.Text(),
	}
	if !ctrl.pendingRemovals.saveValidatedBatch(key, []int64{problemID}) {
		ctrl.editInteractionResponse(event.Interaction, problemRemovalUnavailable)
		return
	}

	response := discord.BuildProblemRemovalPreview(
		preview,
		interactiontransport.ProblemRemovalConfirmID(key.batchID),
		interactiontransport.ProblemRemovalCancelID(key.batchID),
	)
	if _, err := ctrl.responder.EditResponse(event.Interaction, response); err != nil {
		ctrl.pendingRemovals.consumeBatch(key)
	}
}

func (ctrl *ProblemInteractionController) HandleConfirmProblemRemovalInteraction(event *discordgo.InteractionCreate) {
	ctrl.handleProblemRemovalAction(event, true)
}

func (ctrl *ProblemInteractionController) HandleCancelProblemRemovalInteraction(event *discordgo.InteractionCreate) {
	ctrl.handleProblemRemovalAction(event, false)
}

func (ctrl *ProblemInteractionController) handleProblemRemovalAction(event *discordgo.InteractionCreate, confirm bool) {
	if !ctrl.validateProblemRemovalEvent(event, discordgo.InteractionMessageComponent) {
		return
	}

	data := event.MessageComponentData()
	if data.ComponentType != discordgo.ButtonComponent {
		ctrl.responder.RespondPrivately(event.Interaction, "Invalid problem removal button.")
		return
	}

	parseID := interactiontransport.ParseProblemRemovalCancelID
	if confirm {
		parseID = interactiontransport.ParseProblemRemovalConfirmID
	}

	confirmationID, err := parseID(data.CustomID)
	if err != nil {
		ctrl.responder.RespondPrivately(event.Interaction, err.Error())
		return
	}

	key := validatedBatchCacheKey{
		guildID: event.GuildID,
		userID:  event.Member.User.ID,
		batchID: confirmationID,
	}
	if _, available := ctrl.pendingRemovals.loadValidatedBatch(key); !available {
		ctrl.responder.RespondPrivately(event.Interaction, problemRemovalUnavailable)
		return
	}

	if err := ctrl.responder.Respond(event.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate}); err != nil {
		return
	}

	problemIDs, claimed := ctrl.pendingRemovals.consumeBatch(key)
	if !claimed {
		return
	}

	content := "Problem removal cancelled. No changes were applied."
	if confirm {
		problemID := problemIDs[0]
		actor := services.UserActor(event.Member.User.ID)
		if err := ctrl.services.Problem.RemoveProblemFromClass(actor, problemID); err != nil {
			content = adapters.MapError(err, adapters.ActionMentorRemoveProblem, problemID)
		} else {
			content = fmt.Sprintf("Deleted problem `%d` and its recorded ACs.", problemID)
			logger.Log.Infof("Removed problem: actor=%s problem=%d", event.Member.User.ID, problemID)
		}
	}

	if _, err := ctrl.responder.EditResponse(event.Interaction, discord.BuildProblemRemovalResult(content)); err != nil {
		logger.Log.Errorf("Could not deliver problem removal result: confirmation=%s error=%v", confirmationID, err)
	}
}

func (ctrl *ProblemInteractionController) validateProblemRemovalEvent(event *discordgo.InteractionCreate, eventType discordgo.InteractionType) bool {
	if event == nil || event.Interaction == nil || event.Type != eventType {
		return false
	}

	if event.GuildID == "" || ctrl.validateProblemActor(event) != nil {
		ctrl.responder.RespondPrivately(event.Interaction, "Problem removal requires an identified server member.")
		return false
	}
	return true
}
