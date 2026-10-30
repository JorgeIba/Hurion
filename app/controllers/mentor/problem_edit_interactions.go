package mentor

import (
	"errors"
	"strconv"
	"strings"

	"github.com/JorgeIba/Hurion/app/controllers/adapters"
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
)

// HandleEditProblemInteraction opens the current problem or a failed draft for the selected destination.
func (ctrl *ProblemInteractionController) HandleEditProblemInteraction(event *discordgo.InteractionCreate) {
	// Opening a modal must be the initial response; lookups cannot be deferred.
	if err := ctrl.validateProblemActor(event); err != nil {
		ctrl.responder.RespondPrivately(event.Interaction, err.Error())
		return
	}

	problemID, classCode, err := parseProblemEditOptions(event.ApplicationCommandData())
	if err != nil {
		ctrl.responder.RespondPrivately(event.Interaction, err.Error())
		return
	}

	problem, err := ctrl.services.Problem.GetProblemByID(problemID)
	if err != nil {
		ctrl.responder.RespondPrivately(event.Interaction, adapters.MapError(err, adapters.ActionMentorUpdateProblem, problemID, classCode, ""))
		return
	}

	destination, err := ctrl.resolveEditProblemClass(problem.ClassID, classCode)
	if err != nil {
		ctrl.responder.RespondPrivately(event.Interaction, err.Error())
		return
	}

	key := editProblemDraftCacheKey{
		guildID:   event.GuildID,
		userID:    event.Member.User.ID,
		problemID: problemID,
		classCode: destination.ClassCode,
	}
	draft, available := ctrl.editProblemDrafts.loadFailedDraft(key)
	modal := mapReplaceProblemParamsToEditProblemModal(services.MapProblemToReplaceProblemParams(problem, destination.ClassCode))
	if available {
		modal.ProblemModalFields = draft.ProblemModalFields
	}

	response, err := interactiontransport.BuildEditProblemModalResponse(modal)

	if err != nil {
		ctrl.responder.RespondPrivately(event.Interaction, "Could not open the editor: "+err.Error())
		return
	}

	// As in add, consume recovered values only when Discord accepts the form.
	if err := ctrl.responder.Respond(event.Interaction, response); err == nil && available {
		ctrl.editProblemDrafts.consumeDraft(key)
	}
}

// HandleEditProblemModalSubmitInteraction saves the replacement values and reports the result privately.
func (ctrl *ProblemInteractionController) HandleEditProblemModalSubmitInteraction(event *discordgo.InteractionCreate) {
	if err := ctrl.deferProblemInteractionResponse(event.Interaction); err != nil {
		return
	}
	if err := ctrl.validateProblemActor(event); err != nil {
		ctrl.editInteractionResponse(event.Interaction, err.Error())
		return
	}

	modal, err := interactiontransport.ParseEditProblemModal(event.ModalSubmitData())
	if err != nil {
		ctrl.editInteractionResponse(event.Interaction, "Could not read the complete editor. Please start again with /problem edit.")
		return
	}

	key := editProblemDraftCacheKey{guildID: event.GuildID, userID: event.Member.User.ID, problemID: modal.ProblemID, classCode: modal.ClassCode}
	params := mapEditProblemModalToReplaceProblemParams(modal)

	if err := ctrl.services.Problem.ReplaceProblemByProblemID(services.UserActor(event.Member.User.ID), params); err != nil {
		content := adapters.MapError(err, adapters.ActionMentorUpdateProblem, params.ProblemID, modal.ClassCode, params.Name)
		recoverDraft := ctrl.editProblemDrafts.saveFailedDraft(key, modal)
		ctrl.editInteractionResponse(event.Interaction, discord.FormatEditProblemError(content, modal.ProblemID, modal.ClassCode, recoverDraft))
		return
	}
	ctrl.editProblemDrafts.consumeDraft(key)
	ctrl.editInteractionResponse(event.Interaction, discord.FormatUpdatedProblemReceipt(params.ProblemID, modal.ClassCode, params.Name))
}

// resolveEditProblemClass defaults to the problem's saved class.
func (ctrl *ProblemInteractionController) resolveEditProblemClass(currentClassID int64, classCode string) (service_model.Class, error) {
	var selectedClass service_model.Class
	var err error

	if classCode == "" {
		selectedClass, err = ctrl.services.Class.GetClassByClassID(currentClassID)
	} else {
		selectedClass, err = ctrl.services.Class.GetClassByClassCode(classCode)
	}
	if err != nil {
		return service_model.Class{}, errors.New("Could not load the destination class. Check its code and try /problem edit again.")
	}

	return selectedClass, nil
}

func parseProblemEditOptions(command discordgo.ApplicationCommandInteractionData) (int64, string, error) {
	if len(command.Options) != 1 || command.Options[0] == nil {
		return 0, "", errors.New("Use /problem edit problem:<positive ID>.")
	}

	idText, classCode := "", ""
	for _, option := range command.Options[0].Options {
		if option == nil {
			continue
		}
		value, ok := option.Value.(string)
		if !ok {
			return 0, "", errors.New("Problem ID and class must be text.")
		}
		switch option.Name {
		case "problem":
			idText = strings.TrimSpace(value)
		case "class":
			classCode = strings.TrimSpace(value)
		}
	}

	problemID, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || problemID <= 0 {
		return 0, "", errors.New("Problem ID must be a positive integer.")
	}
	return problemID, classCode, nil
}
