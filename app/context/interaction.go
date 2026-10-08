package context

import (
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/bwmarrin/discordgo"
)

type ProblemInteractionController interface {
	HandleEditProblemInteraction(*discordgo.InteractionCreate)
	HandleEditProblemModalSubmitInteraction(*discordgo.InteractionCreate)
	HandleAddProblemInteraction(*discordgo.InteractionCreate)
	HandleCreateProblemModalSubmitInteraction(*discordgo.InteractionCreate)
}

type InteractionContext struct {
	Interaction       *discordgo.InteractionCreate
	Dispatcher        dispatcher.InteractionResponder
	ProblemController ProblemInteractionController
}
