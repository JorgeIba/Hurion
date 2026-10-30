package discordcommands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

const (
	ProblemCommandName          = "problem"
	ProblemAddCommandName       = "add"
	ProblemEditCommandName      = "edit"
	ProblemRemoveCommandName    = "remove"
	ProblemAddBatchCommandName  = "add_batch"
	ProblemEditBatchCommandName = "edit_batch"
	ProblemIDOptionName         = "problem"
)

const (
	ConfirmProblemRemoval   Action = "huron:problem:remove:confirm"
	CancelProblemRemoval    Action = "huron:problem:remove:cancel"
	ProblemEditBatch        Action = "huron:problem:batch:edit"
	ConfirmProblemEditBatch Action = "huron:problem:batch:edit:confirm"
	CancelProblemEditBatch  Action = "huron:problem:batch:edit:cancel"
	ConfirmProblemBatch     Action = "huron:problem:batch:confirm"
	CreateProblem           Action = "huron:problem:create"
	EditProblem             Action = "huron:problem:edit"
	ProblemBatch            Action = "huron:problem:batch"
	CancelProblemBatch      Action = "huron:problem:batch:cancel"
)

func ProblemSlashCommand(classCodeMaxLength int) *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Type:        discordgo.ChatApplicationCommand,
		Name:        ProblemCommandName,
		Description: "Manage class problems",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        ProblemAddCommandName,
				Description: "Start adding a problem",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "class",
						Description: fmt.Sprintf("Class code (up to %d characters); defaults to the class linked to this channel", classCodeMaxLength),
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        ProblemEditCommandName,
				Description: "Edit an existing problem",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:         discordgo.ApplicationCommandOptionString,
						Name:         ProblemIDOptionName,
						Description:  "Problem ID (search by ID or name in this channel's class)",
						Required:     true,
						Autocomplete: true,
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
				Name:        ProblemAddBatchCommandName,
				Description: "Read a problem batch from JSON",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        ProblemEditBatchCommandName,
				Description: "Update supplied problem fields from JSON",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        ProblemRemoveCommandName,
				Description: "Remove a problem and its recorded ACs after confirmation",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:         discordgo.ApplicationCommandOptionString,
						Name:         ProblemIDOptionName,
						Description:  "Problem ID (search by ID or name in this channel's class)",
						Required:     true,
						Autocomplete: true,
					},
				},
			},
		},
	}
}
