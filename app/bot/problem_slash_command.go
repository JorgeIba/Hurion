package bot

import (
	"errors"
	"fmt"
	"strings"

	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
)

const (
	problemCommandName     = "problem"
	problemAddCommandName  = "add"
	problemEditCommandName = "edit"
	batchAddCommandName    = "add_batch"
)

type applicationCommandCreator interface {
	ApplicationCommandCreate(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error)
}

// UpsertProblemSlashCommand registers the guild-scoped /problem command.
// Discord's create endpoint updates an existing command with the same name,
// type, and scope, so running this again keeps the definition current.
func UpsertProblemSlashCommand(session *discordgo.Session, applicationID, guildID string) error {
	if session == nil {
		return errors.New("Discord session is required")
	}

	return upsertProblemSlashCommand(session, applicationID, guildID)
}

func upsertProblemSlashCommand(api applicationCommandCreator, applicationID, guildID string) error {
	return upsertSlashCommand(api, applicationID, guildID, problemSlashCommand())
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

func problemSlashCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Type:        discordgo.ChatApplicationCommand,
		Name:        problemCommandName,
		Description: "Manage class problems",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        problemAddCommandName,
				Description: "Start adding a problem",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "class",
						Description: fmt.Sprintf("Class code (up to %d characters); defaults to the class linked to this channel", service_model.ClassCodeMaxLength),
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        problemEditCommandName,
				Description: "Edit an existing problem",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "problem",
						Description: "Problem ID",
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "class",
						Description: "Destination class; defaults to the problem's current class",
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        batchAddCommandName,
				Description: "Read a problem batch from JSON",
			},
		},
	}
}
