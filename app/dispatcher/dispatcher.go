package dispatcher

import (
	"fmt"
	"time"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/common"
	"github.com/Necroforger/dgwidgets"
	"github.com/bwmarrin/discordgo"
)

const (
	TIME_ACTIVE_PROBLEMS_LIST = time.Minute * 2
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
		if len(msg.Embeds) > 0 && len(msg.Files) > 0 {
			d.Session.ChannelMessageSendComplex(msg.ChannelID, &discordgo.MessageSend{
				Embeds: msg.Embeds,
				Files:  msg.Files,
			})
		}
	case PaginationEmbed:
		paginator := dgwidgets.NewPaginator(msg.Session, msg.ChannelID)
		paginator.Add(msg.Embeds...)

		paginator.SetPageFooters()
		paginator.ColourWhenDone = 0xffff
		paginator.Widget.Timeout = TIME_ACTIVE_PROBLEMS_LIST
		go paginator.Spawn()

	case ConfirmationMessage:
		embed := &discordgo.MessageEmbed{
			Title:       "CONFIRMATION",
			Description: msg.Content,
			Color:       0xff6b35, // Orange
		}

		sentMsg, err := d.Session.ChannelMessageSendEmbedReply(msg.ChannelID, embed, &discordgo.MessageReference{
			MessageID: msg.ReplyTo,
			ChannelID: msg.ChannelID,
		})

		if err == nil {
			d.Session.MessageReactionAdd(msg.ChannelID, sentMsg.ID, common.REACTION_THUMBS_UP)
			d.Session.MessageReactionAdd(msg.ChannelID, sentMsg.ID, common.REACTION_THUMBS_DOWN)
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

func MakeConfirmationMessage(channelID, replyToMessageID, warningMessage string) MessageResponse {
	return MessageResponse{
		Type:      ConfirmationMessage,
		ChannelID: channelID,
		ReplyTo:   replyToMessageID,
		Content:   warningMessage,
	}
}
