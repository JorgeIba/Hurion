package context

import (
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/providers"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/bwmarrin/discordgo"
)

type InteractionContext struct {
	Interaction      *discordgo.InteractionCreate
	Services         *services.Services
	Dispatcher       dispatcher.InteractionResponder
	DiscordValidator providers.DiscordValidator
}
