package dispatcher

import (
	"time"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/common"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
	"github.com/Necroforger/dgwidgets"
	"github.com/bwmarrin/discordgo"
)

const (
	TIME_ACTIVE_PROBLEMS_LIST = time.Minute * 2
)

//go:generate go tool mockgen -source=$GOFILE -destination=mock_dispatcher/mock_dispatcher.go -package=mock_dispatcher
type MessageSender interface {
	Send(msg MessageResponse)
}

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
			d.safeDispatch(response)
		}
	}()
}

func (d *Dispatcher) safeDispatch(response MessageResponse) {
	defer func() {
		if r := recover(); r != nil {
			logger.Log.Errorf("Error dispatching message: %v\n", r)
		}
	}()
	d.dispatch(response)
}

func (d *Dispatcher) Stop() {
	close(d.Output)
}

func (d *Dispatcher) dispatch(msg MessageResponse) {
	allowedMentions := resolveAllowedMentions(msg)

	switch msg.Type {
	case SimpleMessage:
		if _, err := d.Session.ChannelMessageSendComplex(msg.ChannelID, &discordgo.MessageSend{
			Content:         msg.Content,
			AllowedMentions: allowedMentions,
		}); err != nil {
			logger.Log.Errorf("Error sending simple message to channel %s: %v\n", msg.ChannelID, err)
		}

	case ReplyMessage:
		if _, err := d.Session.ChannelMessageSendComplex(msg.ChannelID, &discordgo.MessageSend{
			Content: msg.Content,
			Reference: &discordgo.MessageReference{
				MessageID: msg.ReplyTo,
				ChannelID: msg.ChannelID,
			},
			AllowedMentions: allowedMentions,
		}); err != nil {
			logger.Log.Errorf("Error sending reply to channel %s: %v\n", msg.ChannelID, err)
		}

	case EmbedMessage:
		if _, err := d.Session.ChannelMessageSendComplex(msg.ChannelID, &discordgo.MessageSend{
			Embed:           msg.Embed,
			AllowedMentions: allowedMentions,
		}); err != nil {
			logger.Log.Errorf("Error sending embed to channel %s: %v\n", msg.ChannelID, err)
		}

	case Reaction:
		if msg.ClearReactions {
			if err := d.Session.MessageReactionsRemoveAll(msg.ChannelID, msg.ReplyTo); err != nil {
				logger.Log.Errorf("Error clearing reactions for message %s: %v\n", msg.ReplyTo, err)
			}
		}
		for _, r := range msg.Reactions {
			if err := d.Session.MessageReactionAdd(msg.ChannelID, msg.ReplyTo, r); err != nil {
				logger.Log.Errorf("Error adding reaction %s to message %s: %v\n", r, msg.ReplyTo, err)
			}
		}

	case FileMessage:
		if len(msg.Files) > 0 {
			if _, err := d.Session.ChannelMessageSendComplex(msg.ChannelID, &discordgo.MessageSend{
				Content:         msg.Content,
				Embeds:          msg.Embeds,
				Files:           msg.Files,
				AllowedMentions: allowedMentions,
			}); err != nil {
				logger.Log.Errorf("Error sending file message to channel %s: %v\n", msg.ChannelID, err)
			}
		}
	case PaginationEmbed:
		paginator := dgwidgets.NewPaginator(d.Session, msg.ChannelID)
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

		sentMsg, err := d.Session.ChannelMessageSendComplex(msg.ChannelID, &discordgo.MessageSend{
			Embed: embed,
			Reference: &discordgo.MessageReference{
				MessageID: msg.ReplyTo,
				ChannelID: msg.ChannelID,
			},
			AllowedMentions: allowedMentions,
		})

		if err == nil {
			if err := d.Session.MessageReactionAdd(msg.ChannelID, sentMsg.ID, common.REACTION_THUMBS_UP); err != nil {
				logger.Log.Errorf("Error adding thumbs up reaction to message %s: %v\n", sentMsg.ID, err)
			}
			if err := d.Session.MessageReactionAdd(msg.ChannelID, sentMsg.ID, common.REACTION_THUMBS_DOWN); err != nil {
				logger.Log.Errorf("Error adding thumbs down reaction to message %s: %v\n", sentMsg.ID, err)
			}
		} else {
			logger.Log.Errorf("Error sending confirmation message to channel %s: %v\n", msg.ChannelID, err)
		}

	default:
		logger.Log.Warnf("Unknown message type: %v\n", msg.Type)
	}
}

func (d *Dispatcher) Send(msg MessageResponse) {
	d.Output <- msg
}

func resolveAllowedMentions(msg MessageResponse) *discordgo.MessageAllowedMentions {
	if msg.MentionPolicy == MentionPolicyCustom && msg.AllowedMentions != nil {
		return msg.AllowedMentions
	}

	switch msg.MentionPolicy {
	case MentionPolicyUsers:
		return &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeUsers},
		}
	case MentionPolicyAll:
		return &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{
				discordgo.AllowedMentionTypeUsers,
				discordgo.AllowedMentionTypeRoles,
				discordgo.AllowedMentionTypeEveryone,
			},
		}
	}

	return &discordgo.MessageAllowedMentions{
		Parse: []discordgo.AllowedMentionType{},
	}
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

func MakeFileMessage(channelID, message string, files []*discordgo.File) MessageResponse {
	return MessageResponse{
		Type:      FileMessage,
		ChannelID: channelID,
		Content:   message,
		Files:     files,
	}
}

func MakeEmbedMessage(channelID string, embed *discordgo.MessageEmbed) MessageResponse {
	return MessageResponse{
		Type:      EmbedMessage,
		ChannelID: channelID,
		Embed:     embed,
	}
}
