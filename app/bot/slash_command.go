package bot

import (
	"errors"
	"fmt"
	"strings"

	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
)

type applicationCommandCreator interface {
	ApplicationCommandCreate(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error)
}

// UpsertSlashCommands registers the bot's guild commands in order, stopping at the first failure.
func UpsertSlashCommands(session *discordgo.Session, applicationID, guildID string) error {
	if session == nil {
		return errors.New("Discord session is required")
	}
	return upsertSlashCommands(session, applicationID, guildID)
}

func upsertSlashCommands(api applicationCommandCreator, applicationID, guildID string) error {
	for _, command := range discordcommands.SlashCommands(service_model.ClassCodeMaxLength) {
		if err := upsertSlashCommand(api, applicationID, guildID, command); err != nil {
			return err
		}
	}
	return nil
}

func upsertSlashCommand(api applicationCommandCreator, applicationID, guildID string, command *discordgo.ApplicationCommand) error {
	applicationID = strings.TrimSpace(applicationID)
	if applicationID == "" {
		return errors.New("Discord application ID is required")
	}
	if guildID == "" {
		return errors.New("Discord guild ID is required")
	}

	_, err := api.ApplicationCommandCreate(applicationID, guildID, command)
	if err != nil {
		return fmt.Errorf("upsert guild slash command /%s: %w", command.Name, err)
	}

	return nil
}
