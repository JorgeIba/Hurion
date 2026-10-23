package mentor

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/JorgeIba/Hurion/app/common"
	"github.com/JorgeIba/Hurion/app/context"
	"github.com/JorgeIba/Hurion/app/controllers/adapters"
	controllers_common "github.com/JorgeIba/Hurion/app/controllers/common"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

const (
	messageErrVerifying          = "There was a problem verifying permissions."
	messageErrNotPermissions     = "Bro has not enough permissions (is not mentor)."
	messageErrProblemPermissions = "Problem management requires the mentor or admin role."
	messageErrACPermissions      = "Adding or approving ACs requires the mentor or admin role."
	messageErrUnexpected         = "Unexpected error."
)

func HandlePendingConfirmation(c *context.ContextMessage) {
	messageID := c.MessageReaction.MessageID

	// First check: Is this a pending confirmation message?
	exists, err := c.Services.Solution.ExistsPendingConfirmation(messageID)
	if err != nil {
		logger.Log.Error(err)
		return
	}
	if !exists {
		return
	}

	switch c.MessageReaction.Emoji.Name {
	case common.REACTION_NO:
		pendingConfirmation, err := c.Services.Solution.RejectACProposal(c.Actor(), messageID)
		if err != nil {
			if _, unavailable := errors.AsType[*serviceerr.NotFoundInfoError](err); unavailable {
				return
			}
			if _, denied := errors.AsType[*serviceerr.ForbiddenError](err); denied {
				return
			}
			if _, changed := errors.AsType[*serviceerr.AuthorizationTargetChangedError](err); changed {
				c.Dispatcher.Send(dispatcher.MakeSimpleMessage(c.MessageReaction.ChannelID, adapters.MapError(err, adapters.ActionMentorAddAC)))
				return
			}
			logger.Log.Errorf("Something happened: %v", err)
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.MessageReaction.ChannelID, messageErrUnexpected),
			)
			return
		}

		logger.Log.Infof("Rejected AC by mentor %s: %+v", c.MessageReaction.UserID, pendingConfirmation)
		c.Dispatcher.Send(
			dispatcher.MessageResponse{
				Type:                   dispatcher.Reaction,
				ChannelID:              c.MessageReaction.ChannelID,
				ReplyTo:                messageID,
				ClearSpecificReactions: []string{common.REACTION_YES, common.REACTION_NO},
				Reactions:              []string{common.REACTION_REJECTED},
			},
		)

	case common.REACTION_YES:
		pendingConfirmation, err := c.Services.Solution.AcceptACProposal(c.Actor(), messageID)
		if err != nil {
			if _, unavailable := errors.AsType[*serviceerr.NotFoundInfoError](err); unavailable {
				return
			}
			if _, denied := errors.AsType[*serviceerr.ForbiddenError](err); denied {
				return
			}
			if _, changed := errors.AsType[*serviceerr.AuthorizationTargetChangedError](err); changed {
				c.Dispatcher.Send(dispatcher.MakeSimpleMessage(c.MessageReaction.ChannelID, adapters.MapError(err, adapters.ActionMentorAddAC)))
				return
			}
			logger.Log.Errorf("Something happened: %v", err)

			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.MessageReaction.ChannelID, messageErrUnexpected),
			)
			return
		}

		logger.Log.Infof("Problem AC-ed by mentor %s: %+v", c.MessageReaction.UserID, pendingConfirmation)
		c.Dispatcher.Send(
			dispatcher.MessageResponse{
				Type:                   dispatcher.Reaction,
				ChannelID:              c.MessageReaction.ChannelID,
				ReplyTo:                messageID,
				ClearSpecificReactions: []string{common.REACTION_YES, common.REACTION_NO},
				Reactions:              []string{common.REACTION_ACCEPTED},
			},
		)
	}
}

func HandleAddAC(c *context.ContextMessage) {
	userParam, problemIDs := c.Args[1], c.Args[2]
	userID, ok := controllers_common.ParseUserID(userParam)
	if !ok {
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("bro `%s` is not a valid user mention or ID", userParam)),
		)
		return
	}

	problemID, err := strconv.ParseInt(problemIDs, 10, 64)
	if err != nil {
		logger.Log.Error("Error parsing the ID", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("I couldn't parse `%s` as a problem ID, bro.", problemIDs)),
		)

		return
	}

	if err := c.Services.Solution.CheckManagement(c.Actor(), problemID); err != nil {
		c.Dispatcher.Send(dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionMentorAddAC, userID, problemID)))
		return
	}

	displayName := controllers_common.ResolveDisplayName(c, c.ExecutorGuildID, userID)

	member, err := controllers_common.GetMemberSafe(c, userID)
	if err != nil || member == nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "I couldn't verify the student's roles with Discord, bro. Try again in a moment."),
		)
		return
	}

	member.User.DisplayName = displayName // Override display name to ensure it's correct even if cache is stale
	principal := service_model.Principal{
		User:  member.User,
		Roles: member.Roles,
	}

	if err := c.Services.Solution.AddACByPrincipal(c.Actor(), principal, problemID); err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionMentorAddAC, displayName, problemID)),
		)

		return
	}
	logger.Log.Infof("Problem %d AC-ed by %s (%s) by mentor %s\n", problemID, userID, displayName, c.Executor.UserID)
	c.Dispatcher.Send(
		dispatcher.MessageResponse{
			Type:      dispatcher.Reaction,
			ChannelID: c.Message.ChannelID,
			ReplyTo:   c.Message.ID,
			Reactions: []string{common.REACTION_ACCEPTED},
		},
	)
}

func HandleUndoAC(c *context.ContextMessage) {
	userParam, problemIDs := c.Args[1], c.Args[2]
	userID, ok := controllers_common.ParseUserID(userParam)
	if !ok {
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("bro `%s` is not a valid user mention or ID", userParam)),
		)
		return
	}

	problemID, err := strconv.ParseInt(problemIDs, 10, 64)
	if err != nil {
		logger.Log.Error("Error parsing the ID", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("I couldn't parse `%s` as a problem ID, bro.", problemIDs)),
		)
		return
	}

	if err := c.Services.Solution.CheckUndoAccess(c.Actor(), userID, problemID); err != nil {
		c.Dispatcher.Send(dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionMentorAddAC, userID, problemID)))
		return
	}

	displayName := controllers_common.ResolveDisplayName(c, c.ExecutorGuildID, userID)

	if err = c.Services.Solution.UndoAC(c.Actor(), userID, problemID); err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionMentorUndoAC, displayName, problemID)),
		)
		return
	}

	logger.Log.Infof("Problem %d has been UnAC-ed successfully for user %s (%s) by mentor %s", problemID, userID, displayName, c.Executor.UserID)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf(
			"Problem `%d` has been UnAC-ed successfully for <@%s>",
			problemID,
			userID,
		)),
	)
}

func HandleAddProblem(c *context.ContextMessage) {
	// Parse: !add_problem <class_code> <url> <difficulty or -> <topic or -> <priority or -> <ProblemName...>
	// Args[0] = command, Args[1] = class_code, Args[2] = url, Args[3] = difficulty, Args[4] = topic, Args[5] = priority, Args[6+] = name
	classCode := c.Args[1]
	url := c.Args[2]
	difficultyStr := c.Args[3]
	topicStr := c.Args[4]
	priority, err := parsePriority(c.Args[5])
	if err != nil {
		c.Dispatcher.Send(dispatcher.MakeSimpleMessage(c.Message.ChannelID, err.Error()))
		return
	}
	problemName := strings.Join(c.Args[6:], " ")
	if problemName == "" {
		c.Dispatcher.Send(dispatcher.MakeSimpleMessage(c.Message.ChannelID, "problem name is required"))
		return
	}

	// Parse difficulty (case-insensitive)
	difficulty, err := parseDifficulty(difficultyStr)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, err.Error()),
		)
		return
	}

	// Parse topic
	topic := parseTopic(topicStr)

	problemID, err := c.Services.Problem.CreateProblemIntoClass(c.Actor(), service_model.CreateProblemParams{
		Name:       problemName,
		URL:        url,
		ClassCode:  classCode,
		Difficulty: difficulty,
		Topic:      topic,
		Priority:   priority,
	})

	if err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionMentorAddProblem, classCode, url, problemName)),
		)

		return
	}

	logger.Log.Infof("Problem %s was added correctly to the class %s, Problem ID: `%d`\n", problemName, classCode, problemID)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf(
			"Problem `%s` was added correctly to the class `%s`, Problem ID: `%d`\n",
			problemName,
			classCode,
			problemID),
		),
	)
}

func HandleRemoveProblem(c *context.ContextMessage) error {
	problemIDs := c.Args[1]
	problemID, err := strconv.ParseInt(problemIDs, 10, 64)
	if err != nil {
		logger.Log.Error("Error parsing the ID", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("I couldn't parse `%s` as a problem ID, bro.", problemIDs)),
		)
		return err
	}

	if err = c.Services.Problem.RemoveProblemFromClass(c.Actor(), problemID); err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionMentorRemoveProblem, problemID)),
		)
		return err
	}

	logger.Log.Infof("Problem with ID: %d has been removed from this earthly plane and has passed to the astral plane", problemID)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf(
			"Problem with ID: `%d` has been removed from this earthly plane and has passed to the astral plane",
			problemID)),
	)

	return nil
}

func HandleUpdateProblem(c *context.ContextMessage) {
	problemIDStr := c.Args[1]
	classCode := c.Args[2]
	url := c.Args[3]
	difficultyStr := c.Args[4]
	topicStr := c.Args[5]
	priority, err := parsePriority(c.Args[6])
	if err != nil {
		c.Dispatcher.Send(dispatcher.MakeSimpleMessage(c.Message.ChannelID, err.Error()))
		return
	}
	problemName := strings.Join(c.Args[7:], " ")
	if problemName == "" {
		c.Dispatcher.Send(dispatcher.MakeSimpleMessage(c.Message.ChannelID, "problem name is required"))
		return
	}

	problemID, err := strconv.ParseInt(problemIDStr, 10, 64)
	if err != nil {
		logger.Log.Error("Error parsing the ID", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("I couldn't parse `%s` as a problem ID, bro.", problemIDStr)),
		)
		return
	}

	// Parse difficulty (case-insensitive)
	difficulty, err := parseDifficulty(difficultyStr)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, err.Error()),
		)
		return
	}

	// Parse topic
	topic := parseTopic(topicStr)

	if err = c.Services.Problem.ReplaceProblemByProblemID(c.Actor(), service_model.ReplaceProblemParams{
		ProblemID:  problemID,
		Name:       problemName,
		ClassCode:  classCode,
		URL:        url,
		Difficulty: difficulty,
		Topic:      topic,
		Priority:   priority,
	}); err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionMentorUpdateProblem, problemID, classCode, problemName)),
		)
		return
	}

	logger.Log.Infof("Problem with ID: %d has been updated successfully", problemID)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Problem with ID: `%d` has been updated successfully", problemID)),
	)
}

func parseDifficulty(input string) (*string, error) {
	if input == "-" {
		return nil, nil
	}

	code, err := service_model.ParseDifficulty(input)
	if err != nil {
		return nil, err
	}
	return &code, nil
}

func parseTopic(input string) *string {
	if input == "-" {
		return nil
	}
	return &input
}

func parsePriority(input string) (*string, error) {
	if input == "-" {
		return nil, nil
	}

	code, err := service_model.ParsePriority(input)
	if err != nil {
		return nil, err
	}
	return &code, nil
}
