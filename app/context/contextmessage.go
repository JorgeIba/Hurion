package context

import (
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/validators"
	"github.com/bwmarrin/discordgo"
)

type ContextMessage struct {
	Session          *discordgo.Session
	MessageReaction  *discordgo.MessageReactionAdd
	Message          *discordgo.MessageCreate
	DiscordValidator validators.DiscordValidator
	Dispatcher       *dispatcher.Dispatcher
	Args             []string
	Services         *services.Services

	// ExecutorUserID identifies who is executing the command/action.
	// ExecutorGuildID identifies where (which server) the command is executed.
	// These are the single source of truth for permission checks.
	ExecutorUserID  string
	ExecutorGuildID string
}
