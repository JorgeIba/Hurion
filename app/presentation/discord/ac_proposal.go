package discord

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/JorgeIba/Hurion/app/common"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
)

const (
	acProposalColor             = 0xE5A23B
	acProposalPendingText       = "Pending mentor review"
	acProposalDescriptionFormat = "%s (<@%s>) submitted an AC proposal for:\n**%d — %s**\n\n%s"
	acProposalTitle             = "AC proposal"
	acProposalApproveLabel      = "Approve"
	acProposalRejectLabel       = "Reject"
	acProposalUploadID          = "0"
	acProposalRejectedByText    = " Rejected by "
	acProposalApprovedByText    = " Approved by "
	acProposalRejectedTitle     = "AC proposal rejected"
	acProposalApprovedTitle     = "AC proposal approved"
)

func BuildACProposalWebhookParams(
	proposer service_model.User,
	problem service_model.Problem,
	image []byte,
	contentType string,
	filename string,
	approveID string,
	rejectID string,
) *discordgo.WebhookParams {
	description := fmt.Sprintf(
		acProposalDescriptionFormat,
		inlineCode(proposer.DisplayName), proposer.UserID, problem.ID, problem.Name, acProposalPendingText,
	)

	card := &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{{
			Title:       acProposalTitle,
			Description: description,
			Color:       acProposalColor,
		}},
		Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				CustomID: approveID,
				Label:    acProposalApproveLabel,
				Style:    discordgo.SuccessButton,
			},
			discordgo.Button{
				CustomID: rejectID,
				Label:    acProposalRejectLabel,
				Style:    discordgo.DangerButton,
			},
		}}},
		AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}},
	}

	if len(image) > 0 {
		card.Files = []*discordgo.File{{
			Name:        filename,
			ContentType: contentType,
			Reader:      bytes.NewReader(image),
		}}
		card.Attachments = []*discordgo.MessageAttachment{{ID: acProposalUploadID, Filename: filename}}
	}

	return card
}

func BuildACProposalResolutionEdit(message *discordgo.Message, mentorName string, accepted bool) *discordgo.WebhookEdit {
	status := common.REACTION_REJECTED + acProposalRejectedByText + inlineCode(mentorName)
	title := acProposalRejectedTitle
	if accepted {
		status = common.REACTION_ACCEPTED + acProposalApprovedByText + inlineCode(mentorName)
		title = acProposalApprovedTitle
	}

	components := []discordgo.MessageComponent{}
	edit := &discordgo.WebhookEdit{
		Components:      &components,
		AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}},
	}
	if message == nil {
		content := status
		edit.Content = &content
		return edit
	}

	if len(message.Embeds) > 0 && message.Embeds[0] != nil {
		embeds := append([]*discordgo.MessageEmbed(nil), message.Embeds...)
		updated := *embeds[0]
		updated.Title = title
		updated.Description = strings.TrimSuffix(updated.Description, "\n\n"+acProposalPendingText) + "\n\n" + status
		embeds[0] = &updated
		edit.Embeds = &embeds
	} else {
		content := status
		edit.Content = &content
	}

	if len(message.Attachments) > 0 {
		attachments := make([]*discordgo.MessageAttachment, 0, len(message.Attachments))
		for _, attachment := range message.Attachments {
			if attachment == nil {
				continue
			}
			attachments = append(attachments, &discordgo.MessageAttachment{ID: attachment.ID, Filename: attachment.Filename})
		}
		edit.Attachments = &attachments
	}

	return edit
}
