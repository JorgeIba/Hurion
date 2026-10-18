package discordcommands

import "github.com/bwmarrin/discordgo"

const minimumProblemID float64 = 1

const (
	ACCommandName          = "ac"
	ACAddBatchCommandName  = "add_batch"
	ACSubmitCommandName    = "submit"
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
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        ACProblemIDOptionName,
						Description: "Problem ID",
						Required:    true,
						MinValue:    new(minimumProblemID),
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
		},
	}
}
