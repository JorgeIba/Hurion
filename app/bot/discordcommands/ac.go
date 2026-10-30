package discordcommands

import "github.com/bwmarrin/discordgo"

const (
	ACCommandName          = "ac"
	ACAddBatchCommandName  = "add_batch"
	ACSubmitCommandName    = "submit"
	ACRemoveCommandName    = "remove"
	ACUserOptionName       = "user"
	ACProblemIDOptionName  = "id"
	ACScreenshotOptionName = "screenshot"
)

const (
	ConfirmACBatch    Action = "huron:ac:batch:confirm"
	ACBatch           Action = "huron:ac:batch"
	CancelACBatch     Action = "huron:ac:batch:cancel"
	ApproveACProposal Action = "huron:ac:proposal:approve"
	RejectACProposal  Action = "huron:ac:proposal:reject"
)

func ACSlashCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Type:        discordgo.ChatApplicationCommand,
		Name:        ACCommandName,
		Description: "Manage accepted problems",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        ACSubmitCommandName,
				Description: "Propose a solved problem for mentor approval",
				Options: []*discordgo.ApplicationCommandOption{
					{
						// STRING allows searching by ID or name; suggestions carry only the ID.
						Type:         discordgo.ApplicationCommandOptionString,
						Name:         ACProblemIDOptionName,
						Description:  "Problem ID (search by ID or name in this channel's class)",
						Required:     true,
						Autocomplete: true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionAttachment,
						Name:        ACScreenshotOptionName,
						Description: "Optional screenshot for the mentor",
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        ACAddBatchCommandName,
				Description: "Read an AC batch from JSON",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        ACRemoveCommandName,
				Description: "Remove your AC, or another user's AC as a mentor or admin",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:         discordgo.ApplicationCommandOptionString,
						Name:         ACProblemIDOptionName,
						Description:  "Problem ID (search by ID or name in this channel's class)",
						Required:     true,
						Autocomplete: true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionUser,
						Name:        ACUserOptionName,
						Description: "User whose AC to remove; defaults to yourself",
					},
				},
			},
		},
	}
}
