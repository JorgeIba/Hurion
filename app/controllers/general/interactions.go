package general

import "github.com/JorgeIba/Hurion/app/context"

func HandleUnavailableCommandInteraction(c *context.InteractionContext) {
	command := c.Interaction.ApplicationCommandData()
	c.Dispatcher.RespondPrivately(c.Interaction.Interaction, "That /"+command.Name+" action is not available yet.")
}
