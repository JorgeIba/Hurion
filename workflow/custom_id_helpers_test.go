package workflow

import (
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/bwmarrin/discordgo"
)

func buildCreateProblemModalForTest(modal interactiontransport.CreateProblemModal) (*discordgo.InteractionResponse, error) {
	return interactiontransport.BuildCreateProblemModalResponse(modal)
}
func buildEditProblemModalForTest(modal interactiontransport.EditProblemModal) (*discordgo.InteractionResponse, error) {
	return interactiontransport.BuildEditProblemModalResponse(modal)
}
