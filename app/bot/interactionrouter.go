package bot

import (
	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
	"github.com/JorgeIba/Hurion/app/context"
	"github.com/JorgeIba/Hurion/app/controllers/general"
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/bwmarrin/discordgo"
)

// InteractionRouter handles one Discord command or modal-submission route.
// The bot-level interaction handler owns shared event checks; these routes
// delegate to controllers, which send their responses through the dispatcher.
type InteractionRouter func(*context.InteractionContext)

type autocompleteRouteKey struct {
	command    string
	subcommand string
	option     string
}

var interactionAutocompleteMap = map[autocompleteRouteKey]InteractionRouter{
	{
		command:    discordcommands.ACCommandName,
		subcommand: discordcommands.ACRemoveCommandName,
		option:     discordcommands.ACProblemIDOptionName,
	}: func(c *context.InteractionContext) {
		c.AutocompleteController.HandleProblems(c.Interaction)
	},
	{
		command:    discordcommands.ACCommandName,
		subcommand: discordcommands.ACSubmitCommandName,
		option:     discordcommands.ACProblemIDOptionName,
	}: func(c *context.InteractionContext) {
		c.AutocompleteController.HandleProblems(c.Interaction)
	},
	{
		command:    discordcommands.ProblemCommandName,
		subcommand: discordcommands.ProblemEditCommandName,
		option:     discordcommands.ProblemIDOptionName,
	}: func(c *context.InteractionContext) {
		c.AutocompleteController.HandleProblems(c.Interaction)
	},
}

var interactionCommandMap = map[string]map[string]InteractionRouter{
	discordcommands.ACCommandName: {
		discordcommands.ACRemoveCommandName: func(c *context.InteractionContext) {
			c.ACController.HandleRemoveACInteraction(c.Interaction)
		},
		discordcommands.ACSubmitCommandName: func(c *context.InteractionContext) {
			c.ACController.HandleProposeACInteraction(c.Interaction)
		},
		discordcommands.ACAddBatchCommandName: func(c *context.InteractionContext) {
			c.ACController.HandleAddACBatchInteraction(c.Interaction)
		},
	},
	discordcommands.ProblemCommandName: {
		discordcommands.ProblemEditBatchCommandName: func(c *context.InteractionContext) {
			c.ProblemController.HandleEditProblemBatchInteraction(c.Interaction)
		},
		discordcommands.ProblemAddBatchCommandName: func(c *context.InteractionContext) {
			c.ProblemController.HandleAddProblemBatchInteraction(c.Interaction)
		},
		discordcommands.ProblemEditCommandName: func(c *context.InteractionContext) {
			c.ProblemController.HandleEditProblemInteraction(c.Interaction)
		},
		discordcommands.ProblemAddCommandName: func(c *context.InteractionContext) {
			c.ProblemController.HandleAddProblemInteraction(c.Interaction)
		},
	},
}

var interactionModalMap = map[discordcommands.Action]InteractionRouter{
	discordcommands.ProblemEditBatch: func(c *context.InteractionContext) {
		c.ProblemController.HandleEditProblemBatchModalSubmitInteraction(c.Interaction)
	},
	discordcommands.ProblemBatch: func(c *context.InteractionContext) {
		c.ProblemController.HandleCreateProblemBatchModalSubmitInteraction(c.Interaction)
	},
	discordcommands.ACBatch: func(c *context.InteractionContext) {
		c.ACController.HandleCreateACBatchModalSubmitInteraction(c.Interaction)
	},
	discordcommands.EditProblem: func(c *context.InteractionContext) {
		c.ProblemController.HandleEditProblemModalSubmitInteraction(c.Interaction)
	},
	discordcommands.CreateProblem: func(c *context.InteractionContext) {
		c.ProblemController.HandleCreateProblemModalSubmitInteraction(c.Interaction)
	},
}

var interactionComponentMap = map[discordcommands.Action]InteractionRouter{
	discordcommands.ConfirmProblemEditBatch: func(c *context.InteractionContext) {
		c.ProblemController.HandleConfirmProblemEditBatchInteraction(c.Interaction)
	},
	discordcommands.CancelProblemEditBatch: func(c *context.InteractionContext) {
		c.ProblemController.HandleCancelProblemEditBatchInteraction(c.Interaction)
	},
	discordcommands.ConfirmProblemBatch: func(c *context.InteractionContext) {
		c.ProblemController.HandleConfirmProblemBatchInteraction(c.Interaction)
	},
	discordcommands.ConfirmACBatch: func(c *context.InteractionContext) {
		c.ACController.HandleConfirmACBatchInteraction(c.Interaction)
	},
	discordcommands.CancelProblemBatch: func(c *context.InteractionContext) {
		c.ProblemController.HandleCancelProblemBatchInteraction(c.Interaction)
	},
	discordcommands.CancelACBatch: func(c *context.InteractionContext) {
		c.ACController.HandleCancelACBatchInteraction(c.Interaction)
	},
	discordcommands.ApproveACProposal: func(c *context.InteractionContext) {
		c.ACController.HandleApproveACProposalInteraction(c.Interaction)
	},
	discordcommands.RejectACProposal: func(c *context.InteractionContext) {
		c.ACController.HandleRejectACProposalInteraction(c.Interaction)
	},
}

func hasInteractionRoute(event *discordgo.InteractionCreate) bool {
	switch event.Type {
	case discordgo.InteractionApplicationCommandAutocomplete:
		route := autocompleteRoute(event)
		_, ok := interactionAutocompleteMap[route]
		return ok
	case discordgo.InteractionApplicationCommand:
		_, ok := interactionCommandMap[event.ApplicationCommandData().Name]
		return ok
	case discordgo.InteractionModalSubmit:
		route := interactiontransport.ModalAction(event.ModalSubmitData().CustomID)
		_, ok := interactionModalMap[route]
		return ok
	case discordgo.InteractionMessageComponent:
		route := interactiontransport.ComponentAction(event.MessageComponentData().CustomID)
		_, ok := interactionComponentMap[route]
		return ok
	default:
		return false
	}
}

func autocompleteRoute(event *discordgo.InteractionCreate) autocompleteRouteKey {
	command := event.ApplicationCommandData()
	if len(command.Options) != 1 || command.Options[0] == nil || command.Options[0].Type != discordgo.ApplicationCommandOptionSubCommand {
		return autocompleteRouteKey{}
	}

	focused := interactiontransport.FocusedAutocompleteOption(command.Options[0].Options)
	if focused == nil {
		return autocompleteRouteKey{}
	}

	return autocompleteRouteKey{
		command:    command.Name,
		subcommand: command.Options[0].Name,
		option:     focused.Name,
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

	logger.Log.Infof("Command invoked: /%s %s", command.Name, subcommandName)
	router(c)
}

func routeInteractionAutocomplete(c *context.InteractionContext) {
	route := autocompleteRoute(c.Interaction)
	if router, ok := interactionAutocompleteMap[route]; ok {
		router(c)
	}
}

func routeInteractionModalSubmit(c *context.InteractionContext) {
	route := interactiontransport.ModalAction(c.Interaction.ModalSubmitData().CustomID)
	if router, ok := interactionModalMap[route]; ok {
		logger.Log.Infof("Modal submitted: %s", route)
		router(c)
	}
}

func routeInteractionComponent(c *context.InteractionContext) {
	route := interactiontransport.ComponentAction(c.Interaction.MessageComponentData().CustomID)
	if router, ok := interactionComponentMap[route]; ok {
		logger.Log.Infof("Component invoked: %s", route)
		router(c)
	}
}
