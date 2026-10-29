package mentor

import (
	"fmt"

	"github.com/JorgeIba/Hurion/app/controllers/adapters"
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/bwmarrin/discordgo"
)

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

	if targetUserID != event.Member.User.ID {
		if err := ctrl.authorizeACInteraction(event); err != nil {
			ctrl.editInteractionResponse(event.Interaction, err.Error())
			return
		}
	}

	if err := ctrl.services.Solution.UndoAC(targetUserID, problemID); err != nil {
		logger.Log.Errorf("Could not remove AC: actor=%s user=%s problem=%d error=%v", event.Member.User.ID, targetUserID, problemID, err)
		ctrl.editInteractionResponse(event.Interaction, adapters.MapError(err, adapters.ActionMentorUndoAC, "<@"+targetUserID+">", problemID))
		return
	}

	logger.Log.Infof("Removed AC: actor=%s user=%s problem=%d", event.Member.User.ID, targetUserID, problemID)
	ctrl.editInteractionResponse(event.Interaction, fmt.Sprintf("Removed AC for problem `%d` from <@%s>.", problemID, targetUserID))
}
