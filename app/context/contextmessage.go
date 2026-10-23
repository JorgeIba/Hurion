package context

import (
	"github.com/JorgeIba/Hurion/app/authorization"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/providers"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
)

type ContextMessage struct {
	MessageReaction        *discordgo.MessageReactionAdd
	Message                *discordgo.MessageCreate
	DiscordFetcher         providers.DiscordFetcher
	DiscordMessageModifier providers.DiscordMessageModifier
	Dispatcher             dispatcher.MessageSender
	Args                   []string
	Services               *services.Services

	// Executor identifies who is executing the command/action.
	// It is the single source of truth for permission checks.
	// It is populated at the entry point (bot layer).
	Executor service_model.User

	// ExecutorGuildID identifies where (which server) the command is executed.
	ExecutorGuildID string

	// BotUserID identifies this bot account for validating bot-authored Discord messages.
	BotUserID string
}

// Actor returns the executor identity established by the bot entry point.
func (c *ContextMessage) Actor() authorization.Actor {
	return authorization.Actor{GuildID: c.ExecutorGuildID, UserID: c.Executor.UserID}
}
