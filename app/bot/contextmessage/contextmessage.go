package contextmessage

import (
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/validators"
	"github.com/bwmarrin/discordgo"
)

type ContextMessage struct {
	Session           *discordgo.Session
	MessageReaction   *discordgo.MessageReactionAdd
	Message           *discordgo.MessageCreate
	DatabaseValidator *validators.ValidatorDB
	DiscordValidator  *validators.ValidatorDiscord
	Args              []string
}
