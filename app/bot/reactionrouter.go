package bot

import (
	"fmt"
	"strings"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/common"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	controllers_common "github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/common"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/mentor"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
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
	confirmationMsg, err := c.DiscordFetcher.FetchMessage(c.MessageReaction.ChannelID, c.MessageReaction.MessageID)
	if err != nil {
		return
	}

	if confirmationMsg.ReferencedMessage == nil {
		return
	}

	args := confirmationArgsFromMessage(confirmationMsg)
	if len(args) == 0 {
		setMessageAsError(c)
		return
	}

	// Fetch the original message directly to get complete data
	originalMsg, err := c.DiscordFetcher.FetchMessage(confirmationMsg.ReferencedMessage.ChannelID, confirmationMsg.ReferencedMessage.ID)
	if err != nil {
		logger.Log.Errorf("Failed to fetch original message: %v", err)
		return
	}

	// Check if GuildID is missing and log it
	if originalMsg.GuildID == "" {
		logger.Log.Warnf("Original message missing GuildID, using GuildID from reaction event: %s", c.MessageReaction.GuildID)
		originalMsg.GuildID = c.MessageReaction.GuildID
	}

	// Create new context with original message data
	originalContext := &context.ContextMessage{
		Message:                &discordgo.MessageCreate{Message: originalMsg},
		DiscordValidator:       c.DiscordValidator,
		DiscordFetcher:         c.DiscordFetcher,
		DiscordMessageModifier: c.DiscordMessageModifier,
		Services:               c.Services,
		Dispatcher:             c.Dispatcher,
		Args:                   args,
		Executor:               c.Executor,
		ExecutorGuildID:        c.ExecutorGuildID,
	}

	executeConfirmedCommand(originalContext, c)
}

func confirmationArgsFromMessage(msg *discordgo.Message) []string {
	if msg == nil || len(msg.Embeds) == 0 || msg.Embeds[0].Footer == nil {
		return nil
	}

	commandText, ok := strings.CutPrefix(msg.Embeds[0].Footer.Text, dispatcher.ConfirmationCommandFooterPrefix)
	if !ok {
		return nil
	}

	args := strings.Fields(commandText)
	if len(args) == 0 {
		return nil
	}

	return args
}

func setMessageAsCancelled(c *context.ContextMessage) {
	actor := controllers_common.ResolveDisplayName(c, c.ExecutorGuildID, c.Executor.UserID)
	if actor == "" {
		actor = "Unknown"
	}

	// Update embed as cancelled
	cancelledEmbed := &discordgo.MessageEmbed{
		Title:       "CANCELLED",
		Description: fmt.Sprintf("❌ Operation was cancelled by user _%s_.", actor),
		Color:       0xff0000, // Red
	}
	if err := c.DiscordMessageModifier.EditEmbed(c.MessageReaction.ChannelID, c.MessageReaction.MessageID, cancelledEmbed); err != nil {
		logger.Log.Errorf("Failed to mark message as cancelled: %v", err)
	}

	// Remove reactions
	if err := c.DiscordMessageModifier.RemoveBotReaction(c.MessageReaction.ChannelID, c.MessageReaction.MessageID, common.REACTION_THUMBS_UP); err != nil {
		logger.Log.Errorf("Failed to remove thumbs up reaction: %v", err)
	}
	if err := c.DiscordMessageModifier.RemoveBotReaction(c.MessageReaction.ChannelID, c.MessageReaction.MessageID, common.REACTION_THUMBS_DOWN); err != nil {
		logger.Log.Errorf("Failed to remove thumbs down reaction: %v", err)
	}
}

func setMessageAsExecuted(c *context.ContextMessage, message string) {
	actor := controllers_common.ResolveDisplayName(c, c.ExecutorGuildID, c.Executor.UserID)
	if actor == "" {
		actor = "Unknown"
	}

	// Update embed as Executed
	successEmbed := &discordgo.MessageEmbed{
		Title:       "EXECUTED",
		Description: fmt.Sprintf("✅ %s. Confirmed by _%s_.", message, actor),
		Color:       0x00ff00, // Green
	}
	if err := c.DiscordMessageModifier.EditEmbed(c.MessageReaction.ChannelID, c.MessageReaction.MessageID, successEmbed); err != nil {
		logger.Log.Errorf("Failed to mark message as executed: %v", err)
	}

	// Remove reactions
	if err := c.DiscordMessageModifier.RemoveBotReaction(c.MessageReaction.ChannelID, c.MessageReaction.MessageID, common.REACTION_THUMBS_UP); err != nil {
		logger.Log.Errorf("Failed to remove thumbs up reaction: %v", err)
	}
	if err := c.DiscordMessageModifier.RemoveBotReaction(c.MessageReaction.ChannelID, c.MessageReaction.MessageID, common.REACTION_THUMBS_DOWN); err != nil {
		logger.Log.Errorf("Failed to remove thumbs down reaction: %v", err)
	}
}

func setMessageAsError(c *context.ContextMessage) {
	// Update embed as error
	errorEmbed := &discordgo.MessageEmbed{
		Title:       "ERROR",
		Description: "Something went wrong while processing your request.",
		Color:       0xff0000, // Red
	}
	if err := c.DiscordMessageModifier.EditEmbed(c.MessageReaction.ChannelID, c.MessageReaction.MessageID, errorEmbed); err != nil {
		logger.Log.Errorf("Failed to mark message as error: %v", err)
	}

	// Remove reactions
	if err := c.DiscordMessageModifier.RemoveBotReaction(c.MessageReaction.ChannelID, c.MessageReaction.MessageID, common.REACTION_THUMBS_UP); err != nil {
		logger.Log.Errorf("Failed to remove thumbs up reaction: %v", err)
	}
	if err := c.DiscordMessageModifier.RemoveBotReaction(c.MessageReaction.ChannelID, c.MessageReaction.MessageID, common.REACTION_THUMBS_DOWN); err != nil {
		logger.Log.Errorf("Failed to remove thumbs down reaction: %v", err)
	}
}

func verifyIsConfirmationMessage(c *context.ContextMessage) bool {
	confirmationMsg, err := c.DiscordFetcher.FetchMessage(c.MessageReaction.ChannelID, c.MessageReaction.MessageID)
	if err != nil {
		return false
	}

	if len(confirmationMsg.Embeds) == 0 || confirmationMsg.Embeds[0].Title != "CONFIRMATION" {
		return false
	}

	if confirmationMsg.ReferencedMessage == nil || confirmationMsg.ReferencedMessage.Author == nil {
		return false
	}

	// Verify that the user reacting is the same as the original command author
	if c.MessageReaction.UserID != confirmationMsg.ReferencedMessage.Author.ID {
		return false
	}

	return true
}
