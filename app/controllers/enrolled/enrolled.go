package enrolled

import (
	"fmt"
	"strconv"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/common"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/adapters"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
)

func HandleDeregisterUser(c *context.ContextMessage) error {
	classCode := c.Args[1]
	discordID, discordNickname := c.Message.Author.ID, c.Message.Author.GlobalName

	if err := c.Services.Enrollment.DeregisterUserIDFromClass(discordID, classCode); err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionEnrolledDeregisterUser, discordNickname, classCode)),
		)

		return err
	}

	successMessage := fmt.Sprintf("User `%s` has deregistered from the class `%s` successfully", discordNickname, classCode)
	logger.Log.Info(successMessage)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, successMessage),
	)

	return nil
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

	if err = c.Services.Solution.AddPendingACOfUserID(messageID, discordID, problemID); err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionEnrolledProposeAC, discordNickname, problemID)),
		)

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

	if err = c.Services.Solution.UndoACByProblemAndUserID(problemID, discordID); err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionEnrolledUndoAC, discordNickname, problemID)),
		)

		return
	}

	successMessage := fmt.Sprintf("Problem `%d` has been UnAC-ed successfully by `%s`", problemID, discordNickname)
	logger.Log.Info(successMessage)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, successMessage),
	)
}
