package autocomplete

import (
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/bwmarrin/discordgo"
)

// ProblemController supplies suggestions without running the command's write workflow.
type ProblemController struct {
	services  *services.Services
	responder dispatcher.InteractionResponder
}

func NewProblemController(svcs *services.Services, responder dispatcher.InteractionResponder) *ProblemController {
	return &ProblemController{services: svcs, responder: responder}
}

// HandleProblems suggests existing problems from the class linked to this channel.
// Submission eligibility and mentor authority are checked when the command executes.
func (ctrl *ProblemController) HandleProblems(event *discordgo.InteractionCreate) {
	if event == nil || event.Interaction == nil || event.Type != discordgo.InteractionApplicationCommandAutocomplete {
		return
	}

	choices := []*discordgo.ApplicationCommandOptionChoice{}
	// Every autocomplete event needs its own response, including empty/error results.
	defer func() {
		ctrl.responder.Respond(
			event.Interaction,
			discord.BuildAutocompleteResponse(choices),
		)
	}()

	option := interactiontransport.FocusedAutocompleteOption(event.ApplicationCommandData().Options)
	if option == nil || option.Type != discordgo.ApplicationCommandOptionString {
		return
	}

	query, ok := option.Value.(string)
	if !ok {
		return
	}

	classCode, found, err := ctrl.services.Class.GetClassCodeFromRoom(event.ChannelID)
	if err != nil {
		logger.Log.Warnf("Could not resolve autocomplete class: %v", err)
		return
	}

	if !found {
		return
	}

	problems, err := ctrl.services.Problem.GetProblemsByClass(classCode)
	if err != nil {
		logger.Log.Warnf("Could not load autocomplete problems for class %s: %v", classCode, err)
		return
	}

	choices = discord.ProblemAutocompleteChoices(problems, query)
}
