package mentor

import (
	"errors"
	"strconv"
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
	services          *services.Services
	validator         providers.DiscordValidator
	responder         dispatcher.InteractionResponder
	attachments       *jsonFileTransport
	problemDrafts     *createProblemDraftCache
	editProblemDrafts *editProblemDraftCache
}

func NewProblemInteractionController(svcs *services.Services, validator providers.DiscordValidator, responder dispatcher.InteractionResponder) *ProblemInteractionController {
	return &ProblemInteractionController{
		services:          svcs,
		validator:         validator,
		responder:         responder,
		attachments:       newJSONFileTransport(nil),
		problemDrafts:     newCreateProblemDraftCache(nil),
		editProblemDrafts: newEditProblemDraftCache(nil),
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
	modal := discord.MapCreateProblemParamsToCreateProblemModal(service_model.CreateProblemParams{ClassCode: selectedClass.ClassCode})
	if available {
		modal = draft
	}

	response, err := discord.BuildCreateProblemModalResponse(modal)
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
		// Keep raw input after any failed write so corrections do not require
		// retyping the form, including values that failed validation.
		recoverDraft := ctrl.problemDrafts.saveFailedDraft(key, modal)
		ctrl.editInteractionResponse(event.Interaction, discord.FormatCreateProblemError(content, modal.ClassCode, recoverDraft))
		return
	}

	ctrl.problemDrafts.consumeDraft(key)
	// Response delivery is separate from persistence; a failed receipt must not retry the write.
	ctrl.editInteractionResponse(event.Interaction, discord.FormatCreatedProblemReceipt(problemID, params.ClassCode, params.Name))
}

// HandleEditProblemInteraction opens the current problem or a failed draft for the selected destination.
func (ctrl *ProblemInteractionController) HandleEditProblemInteraction(event *discordgo.InteractionCreate) {
	// Opening a modal must be the initial response; lookups cannot be deferred.
	if err := ctrl.authorizeProblemInteraction(event); err != nil {
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
	modal := discord.MapUpdateProblemParamsToEditProblemModal(services.MapProblemToUpdateProblemParams(problem, destination.ClassCode))
	if available {
		modal.ProblemModalFields = draft.ProblemModalFields
	}

	response, err := discord.BuildEditProblemModalResponse(modal)

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
	if err := ctrl.authorizeProblemInteraction(event); err != nil {
		ctrl.editInteractionResponse(event.Interaction, err.Error())
		return
	}

	modal, err := discord.ParseEditProblemModal(event.ModalSubmitData())
	if err != nil {
		ctrl.editInteractionResponse(event.Interaction, "Could not read the complete editor. Please start again with /problem edit.")
		return
	}

	key := editProblemDraftCacheKey{guildID: event.GuildID, userID: event.Member.User.ID, problemID: modal.ProblemID, classCode: modal.ClassCode}
	params := discord.MapEditProblemModalToUpdateProblemParams(modal)

	if err := ctrl.services.Problem.UpdateProblemByProblemID(params); err != nil {
		content := adapters.MapError(err, adapters.ActionMentorUpdateProblem, params.ProblemID, modal.ClassCode, params.Name)
		recoverDraft := ctrl.editProblemDrafts.saveFailedDraft(key, modal)
		ctrl.editInteractionResponse(event.Interaction, discord.FormatEditProblemError(content, modal.ProblemID, modal.ClassCode, recoverDraft))
		return
	}
	ctrl.editProblemDrafts.consumeDraft(key)
	ctrl.editInteractionResponse(event.Interaction, discord.FormatUpdatedProblemReceipt(params.ProblemID, modal.ClassCode, params.Name))
}

// HandleAddProblemBatchInteraction opens the JSON intake form for problems.
func (ctrl *ProblemInteractionController) HandleAddProblemBatchInteraction(event *discordgo.InteractionCreate) {
	if err := ctrl.authorizeProblemInteraction(event); err != nil {
		ctrl.responder.RespondPrivately(event.Interaction, err.Error())
		return
	}
	ctrl.responder.Respond(event.Interaction, discord.BuildProblemBatchModalResponse())
}

// HandleCreateProblemBatchModalSubmitInteraction parses without preparing or writing.
func (ctrl *ProblemInteractionController) HandleCreateProblemBatchModalSubmitInteraction(event *discordgo.InteractionCreate) {
	if err := ctrl.deferProblemInteractionResponse(event.Interaction); err != nil {
		return
	}
	if err := ctrl.authorizeProblemInteraction(event); err != nil {
		ctrl.editInteractionResponse(event.Interaction, err.Error())
		return
	}
	source, err := discord.ParseProblemBatchModalSource(event.ModalSubmitData())
	if err != nil {
		ctrl.editInteractionResponse(event.Interaction, "Could not read the batch form: "+err.Error())
		return
	}
	input, err := ctrl.attachments.read(source)
	if err != nil {
		ctrl.editInteractionResponse(event.Interaction, "Could not read the batch form: "+err.Error())
		return
	}
	entries, issues := parseProblemBatch(input)
	ctrl.editInteractionResponse(event.Interaction, batchParseReceipt(len(entries), issues))
}

func (ctrl *ProblemInteractionController) deferProblemInteractionResponse(interaction *discordgo.Interaction) error {
	return ctrl.responder.Respond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	})
}

func (ctrl *ProblemInteractionController) editInteractionResponse(interaction *discordgo.Interaction, content string) {
	ctrl.responder.EditResponse(interaction, &discordgo.WebhookEdit{
		Content:         &content,
		AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}},
	})
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
