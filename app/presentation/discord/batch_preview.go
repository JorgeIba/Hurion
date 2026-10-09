package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const (
	ProblemBatchCancelCustomID = "huron:problem:batch:cancel"
	ACBatchCancelCustomID      = "huron:ac:batch:cancel"
	batchActionIDSeparator     = "|"
	BatchUnavailableMessage    = "This batch is no longer available. Submit it again to validate a new batch."
)

func BuildValidatedBatchPreview(content, cancelRoute, batchID string) *discordgo.WebhookEdit {
	components := []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{CustomID: cancelRoute + batchActionIDSeparator + batchID, Label: "Cancel", Style: discordgo.SecondaryButton},
	}}}

	return &discordgo.WebhookEdit{Content: &content, Components: &components, AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}}}
}

func ComponentRouteFromCustomID(customID string) string {
	route, _, _ := strings.Cut(customID, batchActionIDSeparator)
	return route
}

func ParseBatchCancelID(data discordgo.MessageComponentInteractionData, expectedRoute string) (string, error) {
	route, batchID, found := strings.Cut(data.CustomID, batchActionIDSeparator)
	if data.ComponentType != discordgo.ButtonComponent || !found || route != expectedRoute || batchID == "" || strings.TrimSpace(batchID) != batchID || strings.Contains(batchID, batchActionIDSeparator) {
		return "", fmt.Errorf("invalid batch cancellation button")
	}
	return batchID, nil
}

func BuildBatchCancellationResult(cancelled bool) *discordgo.WebhookEdit {
	content := BatchUnavailableMessage
	if cancelled {
		content = "Batch cancelled. No changes were applied."
	}

	components := []discordgo.MessageComponent{}
	return &discordgo.WebhookEdit{Content: &content, Components: &components, AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}}}
}
