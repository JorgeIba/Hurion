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
	DiscordValidator *validators.ValidatorDiscord
	Dispatcher       *dispatcher.Dispatcher
	Args             []string
	Services         *services.Services
}
