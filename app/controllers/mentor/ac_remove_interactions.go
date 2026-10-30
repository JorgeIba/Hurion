package mentor

import (
	"errors"
	"strings"

	appcontext "github.com/JorgeIba/Hurion/app/context"
	"github.com/JorgeIba/Hurion/app/controllers/adapters"
	controllers_common "github.com/JorgeIba/Hurion/app/controllers/common"
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/bwmarrin/discordgo"
)

const messageACRemovalPublicationUncertain = "Removal completed, but I could not confirm whether the public receipt was published. Check this channel for the receipt."

// HandleRemoveACInteraction removes a recorded AC; proposals are handled separately.
func (ctrl *ACInteractionController) HandleRemoveACInteraction(event *discordgo.InteractionCreate) {
	if event == nil || event.Interaction == nil {
		return
	}

	if event.GuildID == "" || event.Member == nil || event.Member.User == nil || event.Member.User.ID == "" {
		ctrl.responder.RespondPrivately(event.Interaction, "AC commands require an identified server member.")
		return
	}

	problemID, targetUserID, err := interactiontransport.ParseACRemoveOptions(event.ApplicationCommandData())
	if err != nil {
		ctrl.responder.RespondPrivately(event.Interaction, err.Error())
		return
	}

	if targetUserID == "" {
		targetUserID = event.Member.User.ID
	}

	if err := ctrl.deferACInteractionResponse(event.Interaction); err != nil {
		return
	}

	if err := ctrl.services.Solution.UndoAC(services.UserActor(event.Member.User.ID), targetUserID, problemID); err != nil {
		logger.Log.Errorf("Could not remove AC: actor=%s user=%s problem=%d error=%v", event.Member.User.ID, targetUserID, problemID, err)
		targetName := "<@" + targetUserID + ">"
		if _, conflict := errors.AsType[*serviceerr.ConflictError](err); conflict {
			name := ""
			if targetUserID == event.Member.User.ID {
				name = controllers_common.GetDisplayNameFromMember(event.Member)
			} else {
				name = controllers_common.ResolveDisplayName(&appcontext.ContextMessage{
					DiscordFetcher: ctrl.fetcher,
					Services:       ctrl.services,
				}, event.GuildID, targetUserID)
			}
			targetName = formatACUndoConflictTarget(name, targetUserID)
		}
		ctrl.editInteractionResponse(event.Interaction, adapters.MapError(err, adapters.ActionMentorUndoAC, targetName, problemID))
		return
	}

	logger.Log.Infof("Removed AC: actor=%s user=%s problem=%d", event.Member.User.ID, targetUserID, problemID)
	problem, lookupErr := ctrl.services.Problem.GetProblemByID(problemID)
	if lookupErr != nil {
		logger.Log.Warnf("Could not load removed AC problem name: problem=%d error=%v", problemID, lookupErr)
	}

	content := discord.FormatRemovedACReceipt(problemID, problem.Name, targetUserID)
	if _, err := ctrl.responder.Publish(event.Interaction, discord.BuildACRemovalMessage(problemID, problem.Name, targetUserID, event.Member.User.ID)); err != nil {
		ctrl.editInteractionResponse(event.Interaction, content+" "+messageACRemovalPublicationUncertain)
		return
	}

	// Remove the ephemeral response to avoid cluttering the channel with a duplicate message.
	if err := ctrl.responder.DeleteResponse(event.Interaction); err != nil {
		ctrl.editInteractionResponse(event.Interaction, content)
	}
}

func formatACUndoConflictTarget(displayName, targetUserID string) string {
	displayName = strings.TrimSpace(displayName)
	// EnsureUser stores "Unknown" as a placeholder when no Discord name is available.
	if displayName == "" || displayName == targetUserID || displayName == "Unknown" {
		return "<@" + targetUserID + ">"
	}

	safeName := strings.NewReplacer("`", "'", "\r", " ", "\n", " ").Replace(displayName)
	return "`" + safeName + "`"
}
