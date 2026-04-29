package enrolled

import (
	"fmt"
	"strconv"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/common"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/adapters"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
)

func HandleDeregisterUser(c *context.ContextMessage) error {
	classCode := c.Args[1]
	user := c.Executor

	if err := c.Services.Enrollment.DeregisterUserIDFromClass(user.UserID, classCode); err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionEnrolledDeregisterUser, user.DisplayName, classCode)),
		)

		return err
	}

	successMessage := fmt.Sprintf("User `%s` has deregistered from the class `%s` successfully", user.DisplayName, classCode)
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

	user := c.Executor
	member, err := c.DiscordFetcher.GetCachedMember(c.ExecutorGuildID, user.UserID)
	if err != nil || member == nil {
		logger.Log.Error("Error fetching member roles or member is nil", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "I couldn't verify your roles with Discord, bro."),
		)
		return
	}

	principal := service_model.Principal{
		User:  user,
		Roles: member.Roles,
	}

	messageID := c.Message.ID
	if err := c.Services.Solution.AddPendingACByPrincipal(messageID, principal, problemID); err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionEnrolledProposeAC, user.DisplayName, problemID)),
		)

		return
	}

	logger.Log.Infof("Proposing AC: user %s, problem %d", user.UserID, problemID)
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

	user := c.Executor
	if err = c.Services.Solution.UndoAC(user.UserID, problemID); err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionEnrolledUndoAC, user.DisplayName, problemID)),
		)

		return
	}

	successMessage := fmt.Sprintf("Problem `%d` has been UnAC-ed successfully by `%s`", problemID, user.DisplayName)
	logger.Log.Info(successMessage)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, successMessage),
	)
}
