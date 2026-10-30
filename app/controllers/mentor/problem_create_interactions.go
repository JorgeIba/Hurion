package mentor

import (
	"errors"
	"strings"

	"github.com/JorgeIba/Hurion/app/controllers/adapters"
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
)

// HandleAddProblemInteraction opens a form for the explicit class or the channel's class.
func (ctrl *ProblemInteractionController) HandleAddProblemInteraction(event *discordgo.InteractionCreate) {
	// Opening a modal must be the initial response within Discord's three-second
	// deadline. The class lookup must finish within that window because a modal
	// cannot follow a deferred reply. Management permission is checked on submit.
	if err := ctrl.validateProblemActor(event); err != nil {
		ctrl.responder.RespondPrivately(event.Interaction, err.Error())
		return
	}
	classCode, err := parseProblemCreateOptions(event.ApplicationCommandData())
	if err != nil {
		ctrl.responder.RespondPrivately(event.Interaction, err.Error())
		return
	}
	selectedClass, err := ctrl.resolveCreateProblemClass(event.ChannelID, classCode)
	if err != nil {
		ctrl.responder.RespondPrivately(event.Interaction, err.Error())
		return
	}

	key := createProblemDraftCacheKey{
		guildID:   event.GuildID,
		userID:    event.Member.User.ID,
		classCode: selectedClass.ClassCode,
	}
	draft, available := ctrl.problemDrafts.loadFailedDraft(key)
	modal := mapCreateProblemParamsToCreateProblemModal(service_model.CreateProblemParams{ClassCode: selectedClass.ClassCode})
	if available {
		modal = draft
	}

	response, err := interactiontransport.BuildCreateProblemModalResponse(modal)
	if err != nil {
		logger.Log.Warnf("Failed to build problem form: %v", err)
		ctrl.responder.RespondPrivately(event.Interaction, "Could not open the form: "+err.Error())
		return
	}
	// Keep the draft if Discord did not accept the form, so the user can retry.
	if err := ctrl.responder.Respond(event.Interaction, response); err == nil && available {
		ctrl.problemDrafts.consumeDraft(key)
	}
}

// HandleCreateProblemModalSubmitInteraction saves the submitted problem and reports its identity.
func (ctrl *ProblemInteractionController) HandleCreateProblemModalSubmitInteraction(event *discordgo.InteractionCreate) {
	// Acknowledge before live permission checks and persistence. Failed or duplicate
	// acknowledgements stop here, before any write can run.
	if err := ctrl.deferProblemInteractionResponse(event.Interaction); err != nil {
		return
	}
	if err := ctrl.validateProblemActor(event); err != nil {
		ctrl.editInteractionResponse(event.Interaction, err.Error())
		return
	}
	modal, err := interactiontransport.ParseCreateProblemModal(event.ModalSubmitData())
	if err != nil {
		logger.Log.Errorf("Failed to read problem form submission: %v", err)
		ctrl.editInteractionResponse(event.Interaction, "Could not read the complete form. Please start again with /problem add.")
		return
	}

	key := createProblemDraftCacheKey{
		guildID:   event.GuildID,
		userID:    event.Member.User.ID,
		classCode: modal.ClassCode,
	}
	params := mapCreateProblemModalToCreateProblemParams(modal)

	problemID, err := ctrl.services.Problem.CreateProblemIntoClass(services.UserActor(event.Member.User.ID), params)
	if err != nil {
		logger.Log.Errorf("Problem creation failed: actor=%s class=%s error=%v", event.Member.User.ID, params.ClassCode, err)
		content := adapters.MapError(err, adapters.ActionMentorAddProblem, params.ClassCode, params.URL, params.Name)
		// Keep raw input after any failed write so corrections do not require
		// retyping the form, including values that failed validation.
		recoverDraft := ctrl.problemDrafts.saveFailedDraft(key, modal)
		ctrl.editInteractionResponse(event.Interaction, discord.FormatCreateProblemError(content, modal.ClassCode, recoverDraft))
		return
	}

	logger.Log.Infof("Problem created: actor=%s class=%s problem=%d", event.Member.User.ID, params.ClassCode, problemID)
	ctrl.problemDrafts.consumeDraft(key)
	// Response delivery is separate from persistence; a failed receipt must not retry the write.
	ctrl.editInteractionResponse(event.Interaction, discord.FormatCreatedProblemReceipt(problemID, params.ClassCode, params.Name))
}

// resolveCreateProblemClass gives the explicit class option precedence over the
// channel mapping. Lookup failures are logged; returned errors are user-facing.
func (ctrl *ProblemInteractionController) resolveCreateProblemClass(channelID, classCode string) (service_model.Class, error) {
	if classCode == "" {
		resolvedCode, found, err := ctrl.services.Class.GetClassCodeFromRoom(channelID)
		if err != nil {
			logger.Log.Warnf("Failed to resolve problem form class: %v", err)
			return service_model.Class{}, errors.New("Could not load the class for this channel. Please try /problem add with the class option.")
		}
		if !found {
			return service_model.Class{}, errors.New("This channel has no linked class. Please use /problem add class:<class code>.")
		}
		classCode = resolvedCode
	}

	selectedClass, err := ctrl.services.Class.GetClassByClassCode(classCode)
	if err != nil {
		if _, invalid := errors.AsType[*serviceerr.ValidationError](err); invalid {
			return service_model.Class{}, errors.New(adapters.MapError(err, adapters.ActionMentorAddProblem))
		}
		logger.Log.Warnf("Failed to load selected problem class: %v", err)
		return service_model.Class{}, errors.New("Could not load the selected class. Please check the class code and try /problem add again.")
	}

	return selectedClass, nil
}

func parseProblemCreateOptions(command discordgo.ApplicationCommandInteractionData) (string, error) {
	if len(command.Options) != 1 || command.Options[0] == nil {
		return "", errors.New("Use /problem add class:<class code>.")
	}

	classCode := ""
	for _, option := range command.Options[0].Options {
		if option == nil || option.Name != "class" {
			continue
		}

		value, ok := option.Value.(string)
		if !ok {
			return "", errors.New("Class must be text.")
		}
		classCode = strings.TrimSpace(value)
	}
	return classCode, nil
}
