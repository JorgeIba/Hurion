package response

import "github.com/bwmarrin/discordgo"

type MessageResponse struct {
	Content string
	Embed   *discordgo.MessageEmbed
	Error   error
}
