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
	acProposalPendingColor  = 0xFEE75C
	acProposalApprovedColor = 0x57F287
	acProposalRejectedColor = 0xED4245
	// Retained to remove the pending label when resolving cards posted before this layout.
	acProposalPendingText       = "Pending mentor review"
	acProposalDescriptionFormat = "<@%s> submitted an AC proposal for: **[%d: %s](%s)**"
	acProposalTitle             = "AC?"
	acProposalRejectEmoji       = "✖️"
	acProposalUploadID          = "0"
	acProposalRejectedByText    = " Rejected by "
	acProposalApprovedByText    = " Approved by "
	acProposalRejectedTitle     = "AC Rejected"
	acProposalApprovedTitle     = "AC Approved"
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
		proposer.UserID, problem.ID, problem.Name, problem.URL,
	)

	card := &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{{
			Title:       acProposalTitle,
			Description: description,
			Color:       acProposalPendingColor,
		}},
		Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				CustomID: approveID,
				Emoji:    &discordgo.ComponentEmoji{Name: common.REACTION_YES},
				Style:    discordgo.SuccessButton,
			},
			discordgo.Button{
				CustomID: rejectID,
				Emoji:    &discordgo.ComponentEmoji{Name: acProposalRejectEmoji},
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
	color := acProposalRejectedColor
	if accepted {
		status = common.REACTION_ACCEPTED + acProposalApprovedByText + inlineCode(mentorName)
		title = acProposalApprovedTitle
		color = acProposalApprovedColor
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
		updated.Color = color
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
