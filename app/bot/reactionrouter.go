package bot

import (
	"fmt"
	"strings"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/common"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/mentor"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
	"github.com/bwmarrin/discordgo"
)

type ReactionRouter func(c *context.ContextMessage)

var reactionRouterMap = map[string]CommandRouter{
	common.REACTION_NO:          routerPendingConfirmationAC,
	common.REACTION_YES:         routerPendingConfirmationAC,
	common.REACTION_THUMBS_UP:   routerConfirmationReaction,
	common.REACTION_THUMBS_DOWN: routerConfirmationReaction,
}

func routerPendingConfirmationAC(c *context.ContextMessage) {
	mentor.HandlePendingConfirmation(c)
}

func routerConfirmationReaction(c *context.ContextMessage) {
	if !verifyIsConfirmationMessage(c) {
		return
	}

	switch c.MessageReaction.Emoji.Name {
	case common.REACTION_THUMBS_UP:
		handleConfirmationYes(c)
	case common.REACTION_THUMBS_DOWN:
		handleConfirmationNo(c)
	}
}

func handleConfirmationNo(c *context.ContextMessage) {
	setMessageAsCancelled(c)
}

func handleConfirmationYes(c *context.ContextMessage) {
	// Get the confirmation message
	confirmationMsg, err := c.Session.ChannelMessage(c.MessageReaction.ChannelID, c.MessageReaction.MessageID)
	if err != nil {
		return
	}

	if confirmationMsg.ReferencedMessage == nil {
		return
	}

	// Fetch the original message directly to get complete data
	originalMsg, err := c.Session.ChannelMessage(confirmationMsg.ReferencedMessage.ChannelID, confirmationMsg.ReferencedMessage.ID)
	if err != nil {
		logger.Log.Errorf("Failed to fetch original message: %v", err)
		return
	}

	// Parse the original command
	args := strings.Fields(originalMsg.Content)
	if len(args) == 0 {
		return
	}

	// Check if GuildID is missing and log it
	if originalMsg.GuildID == "" {
		logger.Log.Warnf("Original message missing GuildID, using GuildID from reaction event: %s", c.MessageReaction.GuildID)
		originalMsg.GuildID = c.MessageReaction.GuildID
	}

	// Create new context with original message data
	originalContext := &context.ContextMessage{
		Session:          c.Session,
		Message:          &discordgo.MessageCreate{Message: originalMsg},
		DiscordValidator: c.DiscordValidator,
		DiscordFetcher:   c.DiscordFetcher,
		Services:         c.Services,
		Dispatcher:       c.Dispatcher,
		Args:             args,
		ExecutorUserID:   c.ExecutorUserID,
		ExecutorGuildID:  c.ExecutorGuildID,
	}

	executeConfirmedCommand(originalContext, args, c)
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
