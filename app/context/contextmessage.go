package context

import (
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/providers"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services"
	"github.com/bwmarrin/discordgo"
)

type ContextMessage struct {
	Session          *discordgo.Session
	MessageReaction  *discordgo.MessageReactionAdd
	Message          *discordgo.MessageCreate
	DiscordValidator providers.DiscordValidator
	DiscordFetcher   providers.DiscordFetcher
	Dispatcher       *dispatcher.Dispatcher
	Args             []string
	Services         *services.Services

	// ExecutorUserID identifies who is executing the command/action.
	// ExecutorGuildID identifies where (which server) the command is executed.
	// These are the single source of truth for permission checks.
	ExecutorUserID  string
	ExecutorGuildID string
}
