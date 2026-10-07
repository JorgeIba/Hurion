package mentor

import (
	"errors"
	"fmt"
	"strings"

	"github.com/JorgeIba/Hurion/app/controllers/adapters"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/providers"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
)

// ProblemInteractionController shares problem drafts across command and submit events.
type ProblemInteractionController struct {
	services      *services.Services
	validator     providers.DiscordValidator
	responder     dispatcher.InteractionResponder
	problemDrafts *createProblemDraftCache
}

func NewProblemInteractionController(svcs *services.Services, validator providers.DiscordValidator, responder dispatcher.InteractionResponder) *ProblemInteractionController {
	return &ProblemInteractionController{
		services:      svcs,
		validator:     validator,
		responder:     responder,
		problemDrafts: newCreateProblemDraftCache(nil),
	}
}

// HandleAddProblemInteraction opens a form for the explicit class or the channel's class.
func (ctrl *ProblemInteractionController) HandleAddProblemInteraction(event *discordgo.InteractionCreate) {
	// Opening a modal must be the initial response within Discord's three-second
	// deadline. Permission verification calls Discord; it and the class lookup
	// must finish within that window because a modal cannot follow a deferred reply.
	if err := ctrl.authorizeProblemInteraction(event); err != nil {
		ctrl.responder.RespondPrivately(event.Interaction, err.Error())
		return
	}
	selectedClass, err := ctrl.resolveInteractionClass(event)
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
	modal := discord.MapCreateProblemParamsToCreateProblemModal(service_model.CreateProblemParams{ClassCode: selectedClass.ClassCode})
	if available {
		modal = draft
	}

	response, err := discord.BuildCreateProblemModalResponse(modal)
	if err != nil {
		logger.Log.Warnf("Failed to build problem form: %v", err)
		ctrl.responder.RespondPrivately(event.Interaction, "Could not open the form: the class code cannot fit in Discord's form identifier.")
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
	if err := ctrl.responder.Respond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}); err != nil {
		return
	}
	if err := ctrl.authorizeProblemInteraction(event); err != nil {
		ctrl.editInteractionResponse(event.Interaction, err.Error())
		return
	}
	modal, err := discord.ParseCreateProblemModal(event.ModalSubmitData())
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
	params := discord.MapCreateProblemModalToCreateProblemParams(modal)
	problemID, err := ctrl.services.Problem.CreateProblemIntoClass(params)
	if err != nil {
		content := adapters.MapError(err, adapters.ActionMentorAddProblem, params.ClassCode, params.URL, params.Name)
		if _, invalid := errors.AsType[*serviceerr.ValidationError](err); invalid {
			// Keep raw input after validation failure so corrections do not require
			// retyping the form, including the values that failed validation.
			ctrl.problemDrafts.saveFailedDraft(key, modal)
			content += ". Try /problem add class:" + modal.ClassCode + " again."
		} else {
			logger.Log.Errorf("Failed to create problem from form: %v", err)
		}
		ctrl.editInteractionResponse(event.Interaction, discord.FormatCreateProblemError(content, modal.ClassCode))
		return
	}

	ctrl.problemDrafts.consumeDraft(key)
	// Response delivery is separate from persistence; a failed receipt must not retry the write.
	ctrl.editInteractionResponse(event.Interaction, discord.FormatCreatedProblemReceipt(problemID, params.ClassCode, params.Name))
}

func (ctrl *ProblemInteractionController) editInteractionResponse(interaction *discordgo.Interaction, content string) {
	ctrl.responder.EditResponse(interaction, &discordgo.WebhookEdit{
		Content:         &content,
		AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}},
	})
}

// resolveInteractionClass gives the explicit class option precedence over the
// channel mapping. Lookup failures are logged; returned errors are user-facing.
func (ctrl *ProblemInteractionController) resolveInteractionClass(event *discordgo.InteractionCreate) (service_model.Class, error) {
	command := event.ApplicationCommandData()
	commandName := "/" + command.Name + " " + command.Options[0].Name
	classCode := ""
	for _, option := range command.Options[0].Options {
		if option != nil && option.Name == "class" {
			classCode = option.StringValue()
		}
	}
	classCode = strings.TrimSpace(classCode)

	if classCode == "" {
		resolvedCode, found, err := ctrl.services.Class.GetClassCodeFromRoom(event.ChannelID)
		if err != nil {
			logger.Log.Warnf("Failed to resolve problem form class: %v", err)
			return service_model.Class{}, fmt.Errorf("Could not load the class for this channel. Please try %s with the class option.", commandName)
		}
		if !found {
			return service_model.Class{}, fmt.Errorf("This channel has no linked class. Please use %s class:<class code>.", commandName)
		}
		classCode = resolvedCode
	}

	selectedClass, err := ctrl.services.Class.GetClassByClassCode(classCode)
	if err != nil {
		if _, invalid := errors.AsType[*serviceerr.ValidationError](err); invalid {
			return service_model.Class{}, errors.New(adapters.MapError(err, adapters.ActionMentorAddProblem))
		}
		logger.Log.Warnf("Failed to load selected problem class: %v", err)
		return service_model.Class{}, fmt.Errorf("Could not load the selected class. Please check the class code and try %s again.", commandName)
	}

	return selectedClass, nil
}

func (ctrl *ProblemInteractionController) authorizeProblemInteraction(event *discordgo.InteractionCreate) error {
	if event.Member == nil || event.Member.User == nil || event.Member.User.ID == "" {
		return errors.New("Problem commands require an identified server member.")
	}

	allowed, err := canManageProblems(ctrl.validator, event.GuildID, event.Member.User.ID)
	if err != nil {
		logger.Log.Errorf("Failed to verify interaction problem-management permissions: %v", err)
		return errors.New(messageErrVerifying)
	}

	if !allowed {
		return errors.New(messageErrProblemPermissions)
	}
	return nil
}
