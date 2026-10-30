package mentor

import (
	"errors"
	"github.com/JorgeIba/Hurion/app/services"

	"github.com/JorgeIba/Hurion/app/controllers/adapters"
	controllers_common "github.com/JorgeIba/Hurion/app/controllers/common"
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
)

const (
	messageACProposalMemberRequired           = "AC proposals can only be submitted by a server member."
	messageACProposalPreparing                = "Received your proposal. Preparing it for mentor review."
	messageACProposalPublicationFailed        = "Could not confirm publication of your proposal card. Your previous proposal, if any, has been replaced. Check the channel; if the new card is missing, submit the command again."
	messageACProposalPublished                = "Your proposal has been published for mentor review."
	messageACProposalInvalidAction            = "Invalid AC proposal action."
	messageACProposalUnavailable              = "This proposal has already been resolved or replaced."
	messageACProposalDecisionFailed           = "Could not complete the decision. Please try again."
	messageACProposalApprovedCardUpdateFailed = "The proposal was approved, but its public card could not be updated."
	messageACProposalRejectedCardUpdateFailed = "The proposal was rejected, but its public card could not be updated."
)

func (ctrl *ACInteractionController) HandleProposeACInteraction(event *discordgo.InteractionCreate) {
	if event == nil || event.Interaction == nil || event.ID == "" {
		return
	}

	if event.Member == nil || event.Member.User == nil || event.Member.User.ID == "" {
		ctrl.responder.RespondPrivately(event.Interaction, messageACProposalMemberRequired)
		return
	}

	problemID, attachment, err := interactiontransport.ParseACProposalOptions(event.ApplicationCommandData())
	if err != nil {
		ctrl.responder.RespondPrivately(event.Interaction, err.Error())
		return
	}

	if err := ctrl.responder.RespondPrivately(event.Interaction, messageACProposalPreparing); err != nil {
		return
	}

	// The screenshot is transport evidence, not part of AC eligibility. Prepare it
	// before the shared service writes a pending reference.
	var image []byte
	var filename, contentType string
	if attachment != nil {
		image, filename, contentType, err = interactiontransport.DownloadACProposalImage(ctrl.attachmentDownloader, attachment)
		if err != nil {
			ctrl.editInteractionResponse(event.Interaction, err.Error())
			return
		}
	}

	userWithRoles := service_model.UserWithRoles{User: controllers_common.MapMemberToUser(event.Member), Roles: event.Member.Roles}
	problem, err := ctrl.services.Solution.ProposeAC(event.ID, userWithRoles, problemID)
	if err != nil {
		logger.Log.Errorf("Could not prepare AC proposal %s: proposer=%s problem=%d error=%v", event.ID, userWithRoles.UserID, problemID, err)
		ctrl.editInteractionResponse(event.Interaction, adapters.MapError(err, adapters.ActionEnrolledProposeAC, userWithRoles.DisplayName, problemID))
		return
	}

	card := discord.BuildACProposalMessage(
		userWithRoles.User,
		problem,
		image,
		contentType,
		filename,
		interactiontransport.ACProposalApproveID(event.ID), interactiontransport.ACProposalRejectID(event.ID),
	)

	if _, err := ctrl.responder.Publish(event.Interaction, card); err != nil {
		// Publication does not roll back renewal. A later submission can replace this
		// reference again, including when Discord's publication outcome is uncertain.
		logger.Log.Errorf("Could not publish AC proposal %s: %v", event.ID, err)
		ctrl.editInteractionResponse(event.Interaction, messageACProposalPublicationFailed)
		return
	}

	logger.Log.Infof("Published AC proposal %s: proposer=%s problem=%d", event.ID, userWithRoles.UserID, problemID)
	if err := ctrl.responder.DeleteResponse(event.Interaction); err != nil {
		ctrl.editInteractionResponse(event.Interaction, messageACProposalPublished)
	}
}

func (ctrl *ACInteractionController) HandleApproveACProposalInteraction(event *discordgo.InteractionCreate) {
	referenceID, err := interactiontransport.ParseACProposalApproveID(event.MessageComponentData().CustomID)
	if err != nil {
		ctrl.responder.RespondPrivately(event.Interaction, err.Error())
		return
	}
	ctrl.handleACProposalDecision(event, referenceID, true)
}

func (ctrl *ACInteractionController) HandleRejectACProposalInteraction(event *discordgo.InteractionCreate) {
	referenceID, err := interactiontransport.ParseACProposalRejectID(event.MessageComponentData().CustomID)
	if err != nil {
		ctrl.responder.RespondPrivately(event.Interaction, err.Error())
		return
	}
	ctrl.handleACProposalDecision(event, referenceID, false)
}

func (ctrl *ACInteractionController) handleACProposalDecision(event *discordgo.InteractionCreate, referenceID string, accepted bool) {
	if event.MessageComponentData().ComponentType != discordgo.ButtonComponent {
		ctrl.responder.RespondPrivately(event.Interaction, messageACProposalInvalidAction)
		return
	}

	if err := ctrl.responder.Respond(event.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate}); err != nil {
		return
	}

	if err := ctrl.validateACActor(event); err != nil {
		ctrl.responder.FollowupPrivately(event.Interaction, err.Error())
		return
	}

	var pending service_model.PendingConfirmation
	var err error
	if accepted {
		pending, err = ctrl.services.Solution.AcceptACProposal(services.UserActor(event.Member.User.ID), referenceID)
	} else {
		pending, err = ctrl.services.Solution.RejectACProposal(services.UserActor(event.Member.User.ID), referenceID)
	}

	if err != nil {
		if _, denied := errors.AsType[*serviceerr.PermissionDeniedError](err); denied {
			ctrl.responder.FollowupPrivately(event.Interaction, adapters.MapError(err, adapters.ActionMentorAddAC))
			return
		}

		if _, unavailable := errors.AsType[*serviceerr.NotFoundInfoError](err); unavailable {
			ctrl.responder.FollowupPrivately(event.Interaction, messageACProposalUnavailable)
			return
		}
		logger.Log.Errorf("Failed to decide AC proposal %s: %v", referenceID, err)
		ctrl.responder.FollowupPrivately(event.Interaction, messageACProposalDecisionFailed)
		return
	}

	actor := controllers_common.MapMemberToUser(event.Member)
	if _, err := ctrl.responder.EditResponse(event.Interaction, discord.BuildACProposalResolutionEdit(event.Message, actor.DisplayName, accepted)); err != nil {
		logger.Log.Errorf("Failed to update resolved AC proposal %s: %v", referenceID, err)
		message := messageACProposalRejectedCardUpdateFailed
		if accepted {
			message = messageACProposalApprovedCardUpdateFailed
		}
		ctrl.responder.FollowupPrivately(event.Interaction, message)
	}

	logger.Log.Infof("AC proposal %s resolved: accepted=%t mentor=%s student=%s problem=%d", referenceID, accepted, actor.UserID, pending.UserID, pending.ProblemID)
}
