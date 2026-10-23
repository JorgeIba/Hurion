package enrolled

import (
	"fmt"
	"strconv"

	"github.com/JorgeIba/Hurion/app/common"
	"github.com/JorgeIba/Hurion/app/context"
	"github.com/JorgeIba/Hurion/app/controllers/adapters"
	controllers_common "github.com/JorgeIba/Hurion/app/controllers/common"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

func HandleProposeAC(c *context.ContextMessage) {
	problemIDs := c.Args[1]
	problemID, err := strconv.ParseInt(problemIDs, 10, 64)
	if err != nil {
		logger.Log.Error("Error parsing the ID", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("I couldn't parse `%s` as a problem ID, bro.", problemIDs)),
		)
		return
	}

	user := c.Executor
	member, err := controllers_common.GetMemberSafe(c, user.UserID)
	if err != nil || member == nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "I couldn't verify your roles with Discord, bro. Try again in a moment."),
		)
		return
	}
	member.DisplayName = user.DisplayName
	principal := service_model.Principal{
		User:  member.User,
		Roles: member.Roles,
	}

	if _, err = c.Services.Solution.AddPendingACByPrincipal(c.Message.ID, principal, problemID); err != nil {
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
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("I couldn't parse `%s` as a problem ID, bro.", problemIDs)),
		)
		return
	}

	user := c.Executor
	if err = c.Services.Solution.UndoAC(c.Actor(), user.UserID, problemID); err != nil {
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
