package bot

import (
	"github.com/JorgeIba/Hurion/app/context"
	"github.com/JorgeIba/Hurion/app/controllers/general"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/bwmarrin/discordgo"
)

// InteractionRouter handles one Discord command or modal-submission route.
// The bot-level interaction handler owns shared event checks; these routes
// delegate to controllers, which send their responses through the dispatcher.
type InteractionRouter func(*context.InteractionContext)

var interactionCommandMap = map[string]map[string]InteractionRouter{
	acCommandName: {
		batchAddCommandName: func(c *context.InteractionContext) {
			c.ACController.HandleAddACBatchInteraction(c.Interaction)
		},
	},
	problemCommandName: {
		batchAddCommandName: func(c *context.InteractionContext) {
			c.ProblemController.HandleAddProblemBatchInteraction(c.Interaction)
		},
		problemEditCommandName: func(c *context.InteractionContext) {
			c.ProblemController.HandleEditProblemInteraction(c.Interaction)
		},
		problemAddCommandName: func(c *context.InteractionContext) {
			c.ProblemController.HandleAddProblemInteraction(c.Interaction)
		},
	},
}

var interactionModalMap = map[string]InteractionRouter{
	discord.ProblemBatchModalCustomID: func(c *context.InteractionContext) {
		c.ProblemController.HandleCreateProblemBatchModalSubmitInteraction(c.Interaction)
	},
	discord.ACBatchModalCustomID: func(c *context.InteractionContext) {
		c.ACController.HandleCreateACBatchModalSubmitInteraction(c.Interaction)
	},
	discord.EditProblemModalCustomID: func(c *context.InteractionContext) {
		c.ProblemController.HandleEditProblemModalSubmitInteraction(c.Interaction)
	},
	discord.CreateProblemModalCustomID: func(c *context.InteractionContext) {
		c.ProblemController.HandleCreateProblemModalSubmitInteraction(c.Interaction)
	},
}

var interactionComponentMap = map[string]InteractionRouter{
	discord.ProblemBatchCancelCustomID: func(c *context.InteractionContext) {
		c.ProblemController.HandleCancelProblemBatchInteraction(c.Interaction)
	},
	discord.ACBatchCancelCustomID: func(c *context.InteractionContext) {
		c.ACController.HandleCancelACBatchInteraction(c.Interaction)
	},
}

func hasInteractionRoute(event *discordgo.InteractionCreate) bool {
	switch event.Type {
	case discordgo.InteractionApplicationCommand:
		_, ok := interactionCommandMap[event.ApplicationCommandData().Name]
		return ok
	case discordgo.InteractionModalSubmit:
		route := discord.ModalRouteFromCustomID(event.ModalSubmitData().CustomID)
		_, ok := interactionModalMap[route]
		return ok
	case discordgo.InteractionMessageComponent:
		route := discord.ComponentRouteFromCustomID(event.MessageComponentData().CustomID)
		_, ok := interactionComponentMap[route]
		return ok
	default:
		return false
	}
}

func routeInteractionCommand(c *context.InteractionContext) {
	command := c.Interaction.ApplicationCommandData()
	subcommandHandlers := interactionCommandMap[command.Name]

	subcommandName := ""
	if len(command.Options) > 0 && command.Options[0] != nil && command.Options[0].Type == discordgo.ApplicationCommandOptionSubCommand {
		subcommandName = command.Options[0].Name
	}

	router, ok := subcommandHandlers[subcommandName]
	if !ok {
		general.HandleUnavailableCommandInteraction(c)
		return
	}

	router(c)
}

func routeInteractionModalSubmit(c *context.InteractionContext) {
	route := discord.ModalRouteFromCustomID(c.Interaction.ModalSubmitData().CustomID)
	if router, ok := interactionModalMap[route]; ok {
		router(c)
	}
}

func routeInteractionComponent(c *context.InteractionContext) {
	route := discord.ComponentRouteFromCustomID(c.Interaction.MessageComponentData().CustomID)
	if router, ok := interactionComponentMap[route]; ok {
		router(c)
	}
}
