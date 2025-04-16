package dispatcher

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type Dispatcher struct {
	Session *discordgo.Session
	Output  chan MessageResponse
}

func NewDispatcher(session *discordgo.Session, bufferSize int) *Dispatcher {
	return &Dispatcher{
		Session: session,
		Output:  make(chan MessageResponse, bufferSize),
	}
}

func (d *Dispatcher) Start() {
	go func() {
		for response := range d.Output {
			d.dispatch(response)
		}
	}()
}

func (d *Dispatcher) Stop() {
	close(d.Output)
}

func (d *Dispatcher) dispatch(msg MessageResponse) {
	switch msg.Type {
	case SimpleMessage:
		d.Session.ChannelMessageSend(msg.ChannelID, msg.Content)

	case ReplyMessage:
		d.Session.ChannelMessageSendReply(msg.ChannelID, msg.Content, &discordgo.MessageReference{
			MessageID: msg.ReplyTo,
			ChannelID: msg.ChannelID,
		})

	case EmbedMessage:
		d.Session.ChannelMessageSendEmbed(msg.ChannelID, msg.Embed)

	case Reaction:
		for _, r := range msg.Reactions {
			d.Session.MessageReactionAdd(msg.ChannelID, msg.ReplyTo, r)
		}

	case FileMessage:
		if msg.Embed != nil && len(msg.Files) > 0 {
			d.Session.ChannelMessageSendComplex(msg.ChannelID, &discordgo.MessageSend{
				Embed: msg.Embed,
				Files: msg.Files,
			})
		}

	default:
		fmt.Println("Unknown message type:", msg.Type)
	}
}

func (d *Dispatcher) Send(msg MessageResponse) {
	d.Output <- msg
}

func MakeSimpleMessage(channelID, message string) MessageResponse {
	return MessageResponse{
		Type:      SimpleMessage,
		ChannelID: channelID,
		Content:   message,
	}
}
