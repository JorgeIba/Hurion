package mentor

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/JorgeIba/Hurion/app/common"
	"github.com/JorgeIba/Hurion/app/context"
	"github.com/JorgeIba/Hurion/app/controllers/adapters"
	controllers_common "github.com/JorgeIba/Hurion/app/controllers/common"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/providers"
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

	// Check whether the actor can decide AC proposals.
	if allowed, err := checkIfUserCanManageACs(c); err != nil {
		logger.Log.Errorf("Error checking AC permissions: %v", err)
		return
	} else if !allowed {
		logger.Log.Errorln(messageErrACPermissions)
		return
	}

	// Now get the full pending confirmation data
	pendingConfirmation, err := c.Services.Solution.GetPendingConfirmationByMessageID(messageID)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.MessageReaction.ChannelID, messageErrUnexpected),
		)
		return
	}

	switch c.MessageReaction.Emoji.Name {
	case common.REACTION_NO:
		err := c.Services.Solution.RejectACProposal(messageID)
		if err != nil {
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
		err := c.Services.Solution.AcceptACProposal(messageID)
		if err != nil {
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
	if allowed, err := checkIfUserCanManageACs(c); err != nil {
		logger.Log.Errorf("Error checking AC permissions: %v", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, messageErrVerifying),
		)
		return
	} else if !allowed {
		logger.Log.Errorln(messageErrACPermissions)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, messageErrACPermissions),
		)
		return
	}

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

	if err := c.Services.Solution.AddACByPrincipal(principal, problemID); err != nil {
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
	if IsMentor, err := checkIfUserIsMentor(c); err != nil {
		logger.Log.Errorf("Error checking if author is mentor: %v\n", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, messageErrVerifying),
		)
		return
	} else if !IsMentor {
		logger.Log.Errorln(messageErrNotPermissions)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, messageErrNotPermissions),
		)
		return
	}

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

	displayName := controllers_common.ResolveDisplayName(c, c.ExecutorGuildID, userID)

	if err = c.Services.Solution.UndoAC(userID, problemID); err != nil {
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
	if allowed, err := checkIfUserCanManageProblems(c); err != nil {
		logger.Log.Errorf("Failed to verify problem-management permissions: %v", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, messageErrVerifying),
		)
		return
	} else if !allowed {
		logger.Log.Errorln(messageErrProblemPermissions)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, messageErrProblemPermissions),
		)
		return
	}

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

	problemID, err := c.Services.Problem.CreateProblemIntoClass(service_model.CreateProblemParams{
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
	if IsMentor, err := checkIfUserIsMentor(c); err != nil {
		logger.Log.Errorf("Error checking if author is mentor: %v\n", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, messageErrVerifying),
		)
		return err
	} else if !IsMentor {
		logger.Log.Errorln(messageErrNotPermissions)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, messageErrNotPermissions),
		)
		return fmt.Errorf("user is not a mentor")
	}

	problemIDs := c.Args[1]
	problemID, err := strconv.ParseInt(problemIDs, 10, 64)
	if err != nil {
		logger.Log.Error("Error parsing the ID", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("I couldn't parse `%s` as a problem ID, bro.", problemIDs)),
		)
		return err
	}

	if err = c.Services.Problem.RemoveProblemFromClass(problemID); err != nil {
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
	if allowed, err := checkIfUserCanManageProblems(c); err != nil {
		logger.Log.Errorf("Failed to verify problem-management permissions: %v", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, messageErrVerifying),
		)
		return
	} else if !allowed {
		logger.Log.Errorln(messageErrProblemPermissions)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, messageErrProblemPermissions),
		)
		return
	}

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

	if err = c.Services.Problem.ReplaceProblemByProblemID(service_model.ReplaceProblemParams{
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

func checkIfUserIsMentor(c *context.ContextMessage) (bool, error) {
	if c.Executor.UserID == "" {
		return false, fmt.Errorf("no executor user ID available")
	}

	if c.ExecutorGuildID == "" {
		return false, fmt.Errorf("mentor commands can only be used in a server, not in DMs")
	}

	return c.DiscordValidator.IsMentor(c.ExecutorGuildID, c.Executor.UserID)
}

func checkIfUserCanManageProblems(c *context.ContextMessage) (bool, error) {
	if c.Executor.UserID == "" {
		return false, fmt.Errorf("no executor user ID available")
	}
	if c.ExecutorGuildID == "" {
		return false, fmt.Errorf("problem commands can only be used in a server, not in DMs")
	}
	return canManageProblems(c.DiscordValidator, c.ExecutorGuildID, c.Executor.UserID)
}

func checkIfUserCanManageACs(c *context.ContextMessage) (bool, error) {
	if c.Executor.UserID == "" {
		return false, fmt.Errorf("no executor user ID available")
	}
	if c.ExecutorGuildID == "" {
		return false, fmt.Errorf("AC commands can only be used in a server, not in DMs")
	}
	return canManageACs(c.DiscordValidator, c.ExecutorGuildID, c.Executor.UserID)
}

// canManageACs permits mentors and admins to grant or decide ACs.
// A failed role lookup stops authorization.
func canManageACs(validator providers.DiscordValidator, guildID, userID string) (bool, error) {
	isMentor, err := validator.IsMentor(guildID, userID)
	if err != nil || isMentor {
		return isMentor, err
	}
	return validator.IsAdmin(guildID, userID)
}

// canManageProblems permits either configured role. A failed role lookup stops
// authorization; admin is checked only after a successful negative mentor check.
func canManageProblems(validator providers.DiscordValidator, guildID, userID string) (bool, error) {
	isMentor, err := validator.IsMentor(guildID, userID)
	if err != nil || isMentor {
		return isMentor, err
	}
	return validator.IsAdmin(guildID, userID)
}
