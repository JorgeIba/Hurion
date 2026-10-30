package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// BuildACRemovalMessage creates a standalone receipt without a reply reference.
func BuildACRemovalMessage(problemID int64, problemName, targetUserID, actorUserID string) *discordgo.MessageSend {
	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{{
			Title:       "AC Removed",
			Description: FormatRemovedACReceipt(problemID, problemName, targetUserID) + fmt.Sprintf("\n\nRemoved by <@%s>", actorUserID),
			Color:       acProposalRejectedColor,
		}},
		AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}},
	}
}
