package bot

import (
	"fmt"
	"strings"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/common"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/admin"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/mentor"
	"github.com/bwmarrin/discordgo"
)

type ReactionRouter func(c *context.ContextMessage)

var reactionRouterMap = map[string]CommandRouter{
	common.REACTION_NO:          routerPendingConfirmationAC,
	common.REACTION_YES:         routerPendingConfirmationAC,
	common.REACTION_THUMBS_UP:   routerConfirmationReaction,
	common.REACTION_THUMBS_DOWN: routerConfirmationReaction,
}

func routerConfirmationReaction(c *context.ContextMessage) {
	if !verifyIsConfirmationMessage(c) {
		return
	}

	switch c.MessageReaction.Emoji.Name {
	case common.REACTION_THUMBS_UP:
		routerConfirmationThumbsUpReaction(c)
	case common.REACTION_THUMBS_DOWN:
		handleConfirmationNo(c)
	}
}

func routerPendingConfirmationAC(c *context.ContextMessage) {
	mentor.HandlePendingConfirmation(c)
}

func routerConfirmationThumbsUpReaction(c *context.ContextMessage) {
	// Get the confirmation message
	confirmationMsg, err := c.Session.ChannelMessage(c.MessageReaction.ChannelID, c.MessageReaction.MessageID)
	if err != nil {
		return
	}

	// Parse the original command
	args := strings.Fields(confirmationMsg.ReferencedMessage.Content)
	if len(args) == 0 {
		return
	}

	// Create new context with original message data
	originalContext := &context.ContextMessage{
		Session:          c.Session,
		Message:          &discordgo.MessageCreate{Message: confirmationMsg.ReferencedMessage},
		DiscordValidator: c.DiscordValidator,
		Services:         c.Services,
		Dispatcher:       c.Dispatcher,
		Args:             args,
	}

	// Execute the appropriate command
	switch args[0] {
	case "!exterminate":
		if err := admin.HandleExterminate(originalContext); err != nil {
			setMessageAsError(c)
		} else {
			setMessageAsExecuted(c, fmt.Sprintf("User %s has been exterminated", args[1]))
		}
		return
	case "!clear_class":
		if err := admin.HandleClearClass(originalContext); err != nil {
			setMessageAsError(c)
		} else {
			setMessageAsExecuted(c, fmt.Sprintf("Class %s has been cleared", args[1]))
		}
		return
	}
}

func handleConfirmationNo(c *context.ContextMessage) {
	setMessageAsCancelled(c)
}

func setMessageAsCancelled(c *context.ContextMessage) {
	// Update embed as cancelled
	cancelledEmbed := &discordgo.MessageEmbed{
		Title:       "CANCELLED",
		Description: "❌ Operation was cancelled by user.",
		Color:       0xff0000, // Red
	}
	c.Session.ChannelMessageEditEmbed(c.MessageReaction.ChannelID, c.MessageReaction.MessageID, cancelledEmbed)

	// Remove reactions
	c.Session.MessageReactionRemove(c.MessageReaction.ChannelID, c.MessageReaction.MessageID, common.REACTION_THUMBS_UP, c.Session.State.User.ID)
	c.Session.MessageReactionRemove(c.MessageReaction.ChannelID, c.MessageReaction.MessageID, common.REACTION_THUMBS_DOWN, c.Session.State.User.ID)
}

func setMessageAsExecuted(c *context.ContextMessage, message string) {
	// Update embed as Executed
	successEmbed := &discordgo.MessageEmbed{
		Title:       "EXECUTED",
		Description: fmt.Sprintf("✅ %s.", message),
		Color:       0x00ff00, // Green
	}
	c.Session.ChannelMessageEditEmbed(c.MessageReaction.ChannelID, c.MessageReaction.MessageID, successEmbed)

	// Remove reactions
	c.Session.MessageReactionRemove(c.MessageReaction.ChannelID, c.MessageReaction.MessageID, common.REACTION_THUMBS_UP, c.Session.State.User.ID)
	c.Session.MessageReactionRemove(c.MessageReaction.ChannelID, c.MessageReaction.MessageID, common.REACTION_THUMBS_DOWN, c.Session.State.User.ID)
}

func setMessageAsError(c *context.ContextMessage) {
	// Update embed as error
	errorEmbed := &discordgo.MessageEmbed{
		Title:       "ERROR",
		Description: "Something went wrong while processing your request.",
		Color:       0xff0000, // Red
	}
	c.Session.ChannelMessageEditEmbed(c.MessageReaction.ChannelID, c.MessageReaction.MessageID, errorEmbed)

	// Remove reactions
	c.Session.MessageReactionRemove(c.MessageReaction.ChannelID, c.MessageReaction.MessageID, common.REACTION_THUMBS_UP, c.Session.State.User.ID)
	c.Session.MessageReactionRemove(c.MessageReaction.ChannelID, c.MessageReaction.MessageID, common.REACTION_THUMBS_DOWN, c.Session.State.User.ID)
}

func verifyIsConfirmationMessage(c *context.ContextMessage) bool {
	confirmationMsg, err := c.Session.ChannelMessage(c.MessageReaction.ChannelID, c.MessageReaction.MessageID)
	if err != nil {
		return false
	}

	if len(confirmationMsg.Embeds) == 0 || confirmationMsg.Embeds[0].Title != "CONFIRMATION" {
		return false
	}

	if confirmationMsg.ReferencedMessage == nil {
		return false
	}

	// Verify that the user reacting is the same as the original command author
	if c.MessageReaction.UserID != confirmationMsg.ReferencedMessage.Author.ID {
		return false
	}

	return true
}
