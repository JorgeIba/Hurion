package discord

import (
	"fmt"

	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
)

func BuildProblemRemovalPreview(preview service_model.ProblemRemovalPreview, confirmID, cancelID string) *discordgo.WebhookEdit {
	content := fmt.Sprintf("⚠️ Delete **%s** (ID: `%d`) from class `%s`?\nThis permanently deletes the problem and all students' recorded ACs for it.\nThis confirmation expires in 5 minutes.", preview.Problem.Name, preview.Problem.ID, preview.Class.ClassCode)
	components := []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{CustomID: confirmID, Label: "Delete problem", Style: discordgo.DangerButton},
		discordgo.Button{CustomID: cancelID, Label: "Cancel", Style: discordgo.SecondaryButton},
	}}}

	return &discordgo.WebhookEdit{
		Content:         &content,
		Components:      &components,
		AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}},
	}
}

func BuildProblemRemovalResult(content string) *discordgo.WebhookEdit {
	components := []discordgo.MessageComponent{}
	return &discordgo.WebhookEdit{
		Content:         &content,
		Components:      &components,
		AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}},
	}
}
