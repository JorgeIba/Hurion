package enrolled

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/common"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
)

var (
	serviceNotFoundInfoErr = &serviceerr.NotFoundInfoError{}
	serviceConflictErr     = &serviceerr.ConflictError{}
)

func HandleDeregisterUser(c *context.ContextMessage) {
	classCode := c.Args[1]
	discordID, discordNickname := c.Message.Author.ID, c.Message.Author.GlobalName

	if err := c.Services.User.DeregisterUserIDFromClass(discordID, classCode); err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("User `%s` is not registered in class `%s` or class doesn't even exist", discordNickname, classCode)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
			)
		}

		return
	}

	successMessage := fmt.Sprintf("User `%s` has deregistered from the class `%s` successfully", discordNickname, classCode)
	logger.Log.Info(successMessage)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, successMessage),
	)
}

func HandleProposeAC(c *context.ContextMessage) {
	problemIDs := c.Args[1]
	problemID, err := strconv.ParseInt(problemIDs, 10, 64)
	if err != nil {
		logger.Log.Error("Error parsing the ID", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("I couldn't parse `%s` to an integer wtf", problemIDs)),
		)
		return
	}

	messageID, discordID, discordNickname := c.Message.ID, c.Message.Author.ID, c.Message.Author.GlobalName

	if err = c.Services.User.AddPendingACOfUserID(messageID, discordID, problemID); err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Can you check if user `%s` is binded and registered on the same class of problem `%d`?", discordNickname, problemID)),
			)
		case errors.As(err, &serviceConflictErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("User `%s` has already solved problem `%d`", discordNickname, problemID)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
			)
		}

		return
	}

	logger.Log.Info("Adding reactions...")
	c.Dispatcher.Send(
		dispatcher.MessageResponse{
			Type:      dispatcher.Reaction,
			ChannelID: c.Message.ChannelID,
			ReplyTo:   c.Message.ID,
			Reactions: []string{common.REACTION_YES, common.REACTION_NO},
		},
	)
}

func HandleUndoAC(c *context.ContextMessage) {
	problemIDs := c.Args[1]
	problemID, err := strconv.ParseInt(problemIDs, 10, 64)
	if err != nil {
		logger.Log.Error("Error parsing the ID", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("I couldn't parse `%s` to an integer wtf", problemIDs)),
		)
		return
	}

	discordID, discordNickname := c.Message.Author.ID, c.Message.Author.GlobalName

	if err = c.Services.User.UndoACByProblemAndUserID(problemID, discordID); err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Bro `%s` is not able to undo the AC for problem %d", discordNickname, problemID)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
			)
		}

		return
	}

	successMessage := fmt.Sprintf("Problem `%d` has been UnAC-ed successfully by `%s` (or it wasn't even ACed previously, but I didn't want to check it tbh)", problemID, discordNickname)
	logger.Log.Info(successMessage)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, successMessage),
	)
}
