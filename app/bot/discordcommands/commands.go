// Package discordcommands declares the bot's slash commands and interaction actions.
package discordcommands

import "github.com/bwmarrin/discordgo"

// Action identifies an interaction handled by the bot.
type Action string

// SlashCommands lists command definitions in registration order.
// Add new command definitions here. The caller supplies the advertised class limit.
func SlashCommands(classCodeMaxLength int) []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		ProblemSlashCommand(classCodeMaxLength),
		ACSlashCommand(),
	}
}
