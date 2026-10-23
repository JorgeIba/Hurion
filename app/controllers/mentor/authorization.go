package mentor

import (
	"github.com/JorgeIba/Hurion/app/authorization"
	"github.com/bwmarrin/discordgo"
)

func interactionActor(event *discordgo.InteractionCreate) authorization.Actor {
	actor := authorization.Actor{GuildID: event.GuildID}
	if event.Member != nil && event.Member.User != nil {
		actor.UserID = event.Member.User.ID
	}
	return actor
}
