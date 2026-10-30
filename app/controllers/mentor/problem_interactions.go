package mentor

import (
	"errors"

	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/bwmarrin/discordgo"
)

// ProblemInteractionController shares problem drafts across command and submit events.
type ProblemInteractionController struct {
	services             *services.Services
	responder            dispatcher.InteractionResponder
	attachments          *jsonTransport
	validatedBatches     *validatedProblemBatchCache
	validatedEditBatches *validatedProblemEditBatchCache
	problemDrafts        *createProblemDraftCache
	editProblemDrafts    *editProblemDraftCache

	// Reuse the batch cache with one problem ID per entry to keep this change small
	// and retain expiry and atomic consumption. Migrate when a single-value cache
	// is available.
	pendingRemovals *validatedBatchCache[int64]
}

func NewProblemInteractionController(svcs *services.Services, responder dispatcher.InteractionResponder) *ProblemInteractionController {
	return &ProblemInteractionController{
		services:             svcs,
		responder:            responder,
		attachments:          newJSONTransport(nil),
		validatedBatches:     newValidatedProblemBatchCache(nil),
		validatedEditBatches: newValidatedProblemEditBatchCache(nil),
		problemDrafts:        newCreateProblemDraftCache(nil),
		editProblemDrafts:    newEditProblemDraftCache(nil),
		pendingRemovals:      newValidatedBatchCache[int64](nil),
	}
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

func (ctrl *ProblemInteractionController) validateProblemActor(event *discordgo.InteractionCreate) error {
	if event.Member == nil || event.Member.User == nil || event.Member.User.ID == "" {
		return errors.New("Problem commands require an identified server member.")
	}

	return nil
}
