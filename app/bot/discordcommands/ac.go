package discordcommands

import "github.com/bwmarrin/discordgo"

const (
	ACCommandName         = "ac"
	ACAddBatchCommandName = "add_batch"
)

const (
	ConfirmACBatch Action = "huron:ac:batch:confirm"
	ACBatch        Action = "huron:ac:batch"
	CancelACBatch  Action = "huron:ac:batch:cancel"
)

func ACSlashCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Type: discordgo.ChatApplicationCommand, Name: ACCommandName, Description: "Manage accepted problems",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: ACAddBatchCommandName, Description: "Read an AC batch from JSON"},
		},
	}
}
