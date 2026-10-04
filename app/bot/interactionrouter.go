package bot

import "github.com/bwmarrin/discordgo"

// InteractionCommandRouter handles one Discord application-command route.
// The bot-level interaction handler owns shared event checks; these routes own
// command-specific Discord behavior and can later call application handlers.
type InteractionCommandRouter func(*Bot, interactionResponder, *discordgo.InteractionCreate)

var interactionCommandMap = map[string]map[string]InteractionCommandRouter{
	problemCommandName: {
		problemAddCommandName: routerProblemAdd,
	},
}

func hasInteractionCommand(commandName string) bool {
	_, ok := interactionCommandMap[commandName]
	return ok
}

func routeInteractionCommand(bot *Bot, responder interactionResponder, event *discordgo.InteractionCreate) {
	command := event.ApplicationCommandData()
	subcommandHandlers := interactionCommandMap[command.Name]

	subcommandName := ""
	if len(command.Options) > 0 && command.Options[0] != nil && command.Options[0].Type == discordgo.ApplicationCommandOptionSubCommand {
		subcommandName = command.Options[0].Name
	}

	router, ok := subcommandHandlers[subcommandName]
	if !ok {
		bot.respondPrivately(responder, event.Interaction, "That /"+command.Name+" action is not available yet.")
		return
	}

	router(bot, responder, event)
}

func routerProblemAdd(bot *Bot, responder interactionResponder, event *discordgo.InteractionCreate) {
	bot.respondPrivately(responder, event.Interaction, "The /problem add command reached Huron. The form and problem-saving flow are not connected yet.")
}
