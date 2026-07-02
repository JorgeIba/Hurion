package dispatcher

import (
	"fmt"
	"github.com/JorgeIba/Hurion/app/logger"
	"strings"
)

// ConsoleMessageSender is used in offline and script modes to prevent
// sending messages to the real Discord API. It prints outputs to stdout.
type ConsoleMessageSender struct{}

func NewConsoleMessageSender() *ConsoleMessageSender {
	return &ConsoleMessageSender{}
}

func (c *ConsoleMessageSender) Send(msg MessageResponse) {
	switch msg.Type {
	case SimpleMessage:
		fmt.Printf("[BOT MESSAGE -> #%s]: %s\n", msg.ChannelID, msg.Content)
	case ReplyMessage:
		fmt.Printf("[BOT REPLY -> msg:%s in #%s]: %s\n", msg.ReplyTo, msg.ChannelID, msg.Content)
	case EmbedMessage:
		title := "Embed"
		desc := ""
		if msg.Embed != nil {
			if msg.Embed.Title != "" {
				title = msg.Embed.Title
			}
			desc = msg.Embed.Description
		}
		fmt.Printf("[BOT EMBED -> #%s]: [%s] %s\n", msg.ChannelID, title, desc)
	case Reaction:
		fmt.Printf("[BOT REACTION -> msg:%s in #%s]: +%v / -%v\n",
			msg.ReplyTo, msg.ChannelID, msg.Reactions, msg.ClearSpecificReactions)
	case FileMessage:
		var fileNames []string
		for _, f := range msg.Files {
			fileNames = append(fileNames, f.Name)
		}
		fmt.Printf("[BOT FILE -> #%s]: %d files attached (%s)\n", msg.ChannelID, len(msg.Files), strings.Join(fileNames, ", "))
	case PaginationEmbed:
		fmt.Printf("[BOT PAGINATION -> #%s]: Pagination embed sent\n", msg.ChannelID)
	case ConfirmationMessage:
		fmt.Printf("[BOT CONFIRMATION -> msg:%s in #%s]: %s | Args: %s\n",
			msg.ReplyTo, msg.ChannelID, msg.Content, strings.Join(msg.ConfirmationArgs, " "))
	default:
		logger.Log.Warnf("ConsoleMessageSender: Unknown message type: %v\n", msg.Type)
	}
}
