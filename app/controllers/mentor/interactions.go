package mentor

import (
	"fmt"
	"strings"

	"github.com/JorgeIba/Hurion/app/context"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// HandleAddProblemInteraction opens a form for the explicit class or the channel's class.
func HandleAddProblemInteraction(c *context.InteractionContext) {
	event := c.Interaction
	// Opening a modal must be the initial response within Discord's three-second
	// deadline. Permission verification calls Discord; it and the class lookup
	// must finish within that window because a modal cannot follow a deferred reply.
	if !authorizeMentorInteraction(c) {
		return
	}
	selectedClass, err := resolveInteractionClass(c)
	if err != nil {
		c.Dispatcher.RespondPrivately(event.Interaction, err.Error())
		return
	}

	modal := discord.MapCreateProblemParamsToCreateProblemModal(service_model.CreateProblemParams{ClassCode: selectedClass.ClassCode})

	response, err := discord.BuildCreateProblemModalResponse(modal)
	if err != nil {
		logger.Log.Warnf("Failed to build problem form: %v", err)
		c.Dispatcher.RespondPrivately(event.Interaction, "Could not open the form: the class code cannot fit in Discord's form identifier.")
		return
	}
	c.Dispatcher.Respond(event.Interaction, response)
}

// HandleCreateProblemModalSubmitInteraction validates submitted values and sends a preview.
// No problem is saved.
func HandleCreateProblemModalSubmitInteraction(c *context.InteractionContext) {
	event := c.Interaction
	if !authorizeMentorInteraction(c) {
		return
	}
	modal, err := discord.ParseCreateProblemModal(event.ModalSubmitData())
	if err != nil {
		logger.Log.Errorf("Failed to read problem form submission: %v", err)
		c.Dispatcher.RespondPrivately(event.Interaction, "Could not read the complete form. Please start again with /problem add.")
		return
	}

	params, err := service_model.NormalizeAndValidateCreateProblemParams(discord.MapCreateProblemModalToCreateProblemParams(modal))
	if err != nil {
		c.Dispatcher.RespondPrivately(event.Interaction, "Please correct the form: "+err.Error()+". Start again with /problem add.")
		return
	}

	_, err = c.Services.Class.GetClassByClassCode(params.ClassCode)
	if err != nil {
		logger.Log.Warnf("Failed to load submitted problem class: %v", err)
		c.Dispatcher.RespondPrivately(event.Interaction, "Could not load the selected class. Please start again with /problem add.")
		return
	}

	c.Dispatcher.RespondPrivately(event.Interaction, discord.FormatCreateProblemModalPreview(discord.MapCreateProblemParamsToCreateProblemModal(params)))
}

// resolveInteractionClass gives the explicit class option precedence over the
// channel mapping. Lookup failures are logged; returned errors are user-facing.
func resolveInteractionClass(c *context.InteractionContext) (service_model.Class, error) {
	event := c.Interaction
	command := event.ApplicationCommandData()
	commandName := "/" + command.Name + " " + command.Options[0].Name
	classCode := ""
	for _, option := range command.Options[0].Options {
		if option != nil && option.Name == "class" {
			classCode = option.StringValue()
		}
	}
	classCode = strings.TrimSpace(classCode)

	if classCode == "" {
		resolvedCode, found, err := c.Services.Class.GetClassCodeFromRoom(event.ChannelID)
		if err != nil {
			logger.Log.Warnf("Failed to resolve problem form class: %v", err)
			return service_model.Class{}, fmt.Errorf("Could not load the class for this channel. Please try %s with the class option.", commandName)
		}
		if !found {
			return service_model.Class{}, fmt.Errorf("This channel has no linked class. Please use %s class:<class code>.", commandName)
		}
		classCode = resolvedCode
	}

	selectedClass, err := c.Services.Class.GetClassByClassCode(classCode)
	if err != nil {
		logger.Log.Warnf("Failed to load selected problem class: %v", err)
		return service_model.Class{}, fmt.Errorf("Could not load the selected class. Please check the class code and try %s again.", commandName)
	}

	return selectedClass, nil
}

func authorizeMentorInteraction(c *context.InteractionContext) bool {
	event := c.Interaction
	if event.Member == nil || event.Member.User == nil || event.Member.User.ID == "" {
		c.Dispatcher.RespondPrivately(event.Interaction, "Mentor commands require an identified server member.")
		return false
	}

	isMentor, err := c.DiscordValidator.IsMentor(event.GuildID, event.Member.User.ID)
	if err != nil {
		logger.Log.Errorf("Failed to verify interaction mentor permissions: %v", err)
		c.Dispatcher.RespondPrivately(event.Interaction, messageErrVerifying)
		return false
	}

	if !isMentor {
		c.Dispatcher.RespondPrivately(event.Interaction, messageErrNotPermissions)
		return false
	}
	return true
}
