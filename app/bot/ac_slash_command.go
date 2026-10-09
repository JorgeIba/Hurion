package bot

import (
	"errors"

	"github.com/bwmarrin/discordgo"
)

const acCommandName = "ac"

// UpsertACSlashCommand registers the guild-scoped /ac command without changing
// the existing prefix commands.
func UpsertACSlashCommand(session *discordgo.Session, applicationID, guildID string) error {
	if session == nil {
		return errors.New("Discord session is required")
	}
	return upsertACSlashCommand(session, applicationID, guildID)
}

func upsertACSlashCommand(api applicationCommandCreator, applicationID, guildID string) error {
	return upsertSlashCommand(api, applicationID, guildID, acSlashCommand())
}

func acSlashCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Type: discordgo.ChatApplicationCommand, Name: acCommandName, Description: "Manage accepted problems",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: batchAddCommandName, Description: "Read an AC batch from JSON"},
		},
	}
}
