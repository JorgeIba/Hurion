package mentor

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/common"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services"
)

var (
	serviceConflictErr     = &serviceerr.ConflictError{}
	serviceNotFoundInfoErr = &serviceerr.NotFoundInfoError{}
)

const (
	messageErrVerifying      = "There was a problem verifying permissions."
	messageErrNotPermissions = "Bro has not enough permissions (is not mentor)."
	messageErrUnexpectec     = "Unexpected error."
)

func HandleRegisterUser(c *context.ContextMessage) {
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

	userName, classCode := c.Args[1], c.Args[2]

	if err := c.Services.Enrollment.RegisterUsernameIntoClass(userName, classCode); err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Username `%s` or class code `%s` not found", userName, classCode)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, messageErrUnexpectec),
			)
		}

		return
	}

	logger.Log.Infof("User %s has registered in the class %s\n", userName, classCode)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("User `%s` has registered in the class `%s`", userName, classCode)),
	)
}

func HandleDeregisterUser(c *context.ContextMessage) error {
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

	userName, classCode := c.Args[1], c.Args[2]

	if err := c.Services.Enrollment.DeregisterUsernameFromClass(userName, classCode); err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Username `%s` or class code `%s` not found", userName, classCode)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, messageErrUnexpectec),
			)
		}

		return err
	}

	successMessage := fmt.Sprintf("User `%s` has been deregistered from the class %s successfully", userName, classCode)
	logger.Log.Info(successMessage)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, successMessage),
	)
	
	return nil
}

func HandlePendingConfirmation(c *context.ContextMessage) {
	if IsMentor, err := checkIfUserIsMentor(c); err != nil {
		logger.Log.Errorf("Error checking if author is mentor: %v\n", err)
		return
	} else if !IsMentor {
		logger.Log.Errorln(messageErrNotPermissions)
		return
	}

	messageID := c.MessageReaction.MessageID

	pendingConfirmation, err := c.Services.Solution.GetPendingConfirmationByMessageID(messageID)
	if err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):

		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, messageErrUnexpectec),
			)
		}

		return
	}

	switch c.MessageReaction.Emoji.Name {
	case common.REACTION_NO:
		err := c.Services.Solution.RejectACProposal(messageID)
		if err != nil {
			logger.Log.Errorf("Something happened: %v", err)
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.MessageReaction.ChannelID, messageErrUnexpectec),
			)
			return
		}

		logger.Log.Info("Rejected AC", pendingConfirmation)
		c.Dispatcher.Send(
			dispatcher.MessageResponse{
				Type:      dispatcher.ReplyMessage,
				ChannelID: c.MessageReaction.ChannelID,
				ReplyTo:   messageID,
				Content:   "Keep trying lil bro",
			},
		)

	case common.REACTION_YES:
		err := c.Services.Solution.AcceptACProposal(messageID)
		if err != nil {
			logger.Log.Errorf("Something happened: %v", err)
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.MessageReaction.ChannelID, messageErrUnexpectec),
			)
			return
		}

		logger.Log.Infoln("Problem AC-ed", pendingConfirmation)
		c.Dispatcher.Send(
			dispatcher.MessageResponse{
				Type:      dispatcher.ReplyMessage,
				ChannelID: c.MessageReaction.ChannelID,
				ReplyTo:   messageID,
				Content:   fmt.Sprintf("Good job bro, keep going. Problem `%d` has been AC-ed", pendingConfirmation.ProblemID),
			},
		)
	}
}

func HandleAddAC(c *context.ContextMessage) {
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

	userName, problemIDs := c.Args[1], c.Args[2]
	problemID, err := strconv.ParseInt(problemIDs, 10, 64)
	if err != nil {
		logger.Log.Error("Error parsing the ID", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("I couldn't parse `%s` to an integer wtf", problemIDs)),
		)

		return
	}

	if err := c.Services.Solution.AddACToUsername(userName, problemID); err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Username `%s` or problem id `%d` not found, probably you are not binded or registered in the same class problem lives", userName, problemID)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, messageErrUnexpectec),
			)
		}

		return
	}

	logger.Log.Infof("Problem %d AC-ed by %s\n", problemID, userName)
	c.Dispatcher.Send(
		dispatcher.MessageResponse{
			Type:      dispatcher.ReplyMessage,
			ChannelID: c.Message.ChannelID,
			ReplyTo:   c.Message.ID,
			Content:   fmt.Sprintf("Good job lil bro `%s`, keep going. Problem `%d` has been AC-ed", userName, problemID),
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

	problemIDs, userName := c.Args[1], c.Args[2]
	problemID, err := strconv.ParseInt(problemIDs, 10, 64)
	if err != nil {
		logger.Log.Error("Error parsing the ID", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("I couldn't parse `%s` to an integer wtf", problemIDs)),
		)
		return
	}

	if err = c.Services.Solution.UndoACByProblemAndUsername(problemID, userName); err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Username `%s` or problem id `%d` not found", userName, problemID)),
			)
		case errors.As(err, &serviceConflictErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("User `%s` has not solved problem `%d` yet or is not enrolled in the class", userName, problemID)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, messageErrUnexpectec),
			)
		}
		return
	}

	logger.Log.Infof("Problem %d has been UnAC-ed successfully by %s", problemID, userName)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf(
			"Problem `%d` has been UnAC-ed successfully by `%s`",
			problemID,
			userName,
		)),
	)
}

func HandleAddProblem(c *context.ContextMessage) {
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

	// Parse: !add_problem <class_code> <url> <difficulty> <topic> <ProblemName...>
	// Args[0] = command, Args[1] = class_code, Args[2] = url, Args[3] = difficulty, Args[4] = topic, Args[5+] = name
	classCode := c.Args[1]
	url := c.Args[2]
	difficultyStr := c.Args[3]
	topicStr := c.Args[4]
	problemName := strings.Join(c.Args[5:], " ")

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

	problemID, err := c.Services.Problem.CreateProblemIntoClass(services.CreateProblemParams{
		Name:       problemName,
		URL:        url,
		ClassCode:  classCode,
		Difficulty: difficulty,
		Topic:      topic,
	})

	if err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Class code `%s` not found", classCode)),
			)
		case errors.As(err, &serviceConflictErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Problem `%s` already exists on class `%s` with the ID: `%d`", problemName, classCode, problemID)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, messageErrUnexpectec),
			)
		}

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
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("I couldn't parse `%s` to an integer wtf", problemIDs)),
		)
		return err
	}

	if err = c.Services.Problem.RemoveProblemFromClass(problemID); err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Problem id `%d` not found", problemID)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, messageErrUnexpectec),
			)
		}
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

	problemIDStr := c.Args[1]
	classCode := c.Args[2]
	url := c.Args[3]
	difficultyStr := c.Args[4]
	topicStr := c.Args[5]
	problemName := strings.Join(c.Args[6:], " ")

	problemID, err := strconv.ParseInt(problemIDStr, 10, 64)
	if err != nil {
		logger.Log.Error("Error parsing the ID", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("I couldn't parse `%s` to an integer wtf", problemIDStr)),
		)
		return
	}

	classID, err := c.Services.Class.GetClassIDByClassCode(classCode)
	if err != nil {
		logger.Log.Error(err)
		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Class code `%s` not found", classCode)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, messageErrUnexpectec),
			)
		}
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

	if err = c.Services.Problem.UpdateProblemByProblemID(problemID, data.Problem{
		ID:         problemID,
		Name:       problemName,
		ClassID:    classID,
		URL:        url,
		Difficulty: difficulty,
		Topic:      topic,
	}); err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Problem with ID: `%d` not found", problemID)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, messageErrUnexpectec),
			)
		}
		return
	}

	logger.Log.Infof("Problem with ID: %d has been updated successfully", problemID)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Problem with ID: `%d` has been updated successfully", problemID)),
	)
}

var difficultyMap = map[string]string{
	"easy":   "Easy",
	"medium": "Medium",
	"hard":   "Hard",
}

func parseDifficulty(input string) (*string, error) {
	if input == "-" {
		return nil, nil
	}

	// Convert to lowercase for case-insensitive comparison
	lowerInput := strings.ToLower(input)

	if canonical, ok := difficultyMap[lowerInput]; ok {
		return &canonical, nil
	}

	validDifficulties := []string{"easy", "medium", "hard"}
	return nil, fmt.Errorf("invalid difficulty '%s'. Valid options: %s", input, strings.Join(validDifficulties, ", "))
}

func parseTopic(input string) *string {
	if input == "-" {
		return nil
	}
	return &input
}

func checkIfUserIsMentor(c *context.ContextMessage) (bool, error) {
	if c.ExecutorUserID == "" {
		return false, fmt.Errorf("no executor user ID available")
	}

	if c.ExecutorGuildID == "" {
		return false, fmt.Errorf("mentor commands can only be used in a server, not in DMs")
	}

	return c.DiscordValidator.IsMentor(c.ExecutorGuildID, c.ExecutorUserID)
}
