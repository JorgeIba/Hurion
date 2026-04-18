package context

import (
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/providers"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
	"github.com/bwmarrin/discordgo"
)

type ContextMessage struct {
	MessageReaction        *discordgo.MessageReactionAdd
	Message                *discordgo.MessageCreate
	DiscordValidator       providers.DiscordValidator
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
}
