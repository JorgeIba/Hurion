package mentor

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/common"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
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

	if err := c.Services.User.RegisterUsernameIntoClass(userName, classCode); err != nil {
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

func HandleDeregisterUser(c *context.ContextMessage) {
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

	if err := c.Services.User.DeregisterUsernameFromClass(userName, classCode); err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Username %s or class code `%s` not found", userName, classCode)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, messageErrUnexpectec),
			)
		}

		return
	}

	logger.Log.Infof("User %s has registered to the class %s successfully", userName, classCode)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("User %s has registered to the class %s successfully", userName, classCode)),
	)
}

func HandlePendingConfirmation(c *context.ContextMessage) {
	if IsMentor, err := checkIfUserIsMentor(c); err != nil {
		logger.Log.Errorf("Error checking if author is mentor: %v\n", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.MessageReaction.ChannelID, messageErrVerifying),
		)
		return
	} else if !IsMentor {
		logger.Log.Errorln(messageErrNotPermissions)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.MessageReaction.ChannelID, messageErrNotPermissions),
		)
		return
	}

	messageID := c.MessageReaction.MessageID

	pendingConfirmation, err := c.Services.User.GetPendingConfirmationByMessageID(messageID)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.MessageReaction.ChannelID, messageErrUnexpectec),
		)

		return
	}

	switch c.MessageReaction.Emoji.Name {
	case common.REACTION_NO:
		err := c.Services.User.RejectACProposal(messageID)
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
		err := c.Services.User.AcceptACProposal(messageID)
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
				Content:   fmt.Sprintf("Good job bro, keep going. Problem %d has been AC-ed", pendingConfirmation.ProblemID),
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
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("I couldn't parse %s to an integer wtf", problemIDs)),
		)

		return
	}

	if err := c.Services.User.AddACToUsername(userName, problemID); err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Username %s or problem id `%d` not found, probably you are not binded or registered in the same class problem lives", userName, problemID)),
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
			Content:   fmt.Sprintf("Good job lil bro %s, keep going. Problem %d has been AC-ed", userName, problemID),
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
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("I couldn't parse %s to an integer wtf", problemIDs)),
		)
		return
	}

	if err = c.Services.User.UndoACByProblemAndUsername(problemID, userName); err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Username %s or problem id `%d` not found", userName, problemID)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, messageErrUnexpectec),
			)
		}
		return
	}

	logger.Log.Infof("Problem %d has been UnAC-ed successfully by %s (or it wasn't even ACed previously, but I didn't want to check it tbh)", problemID, userName)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf(
			"Problem %d has been UnAC-ed successfully by %s (or it wasn't even ACed previously, but I didn't want to check it tbh)",
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

	classCode, problemName := c.Args[1], strings.Join(c.Args[2:], " ")
	problemID, err := c.Services.User.CreateProblemIntoClass(problemName, classCode)

	if err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Class code `%s` not found", classCode)),
			)
		case errors.As(err, &serviceConflictErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Problem %s already exists on class %s", problemName, classCode)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, messageErrUnexpectec),
			)
		}

		return
	}

	logger.Log.Infof("Problem %s was added correctly to the class %s, Problem ID: %d\n", problemName, classCode, problemID)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf(
			"Problem %s was added correctly to the class %s, Problem ID: %d\n",
			problemName,
			classCode,
			problemID)),
	)
}

func HandleRemoveProblem(c *context.ContextMessage) {
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

	problemIDs := c.Args[1]
	problemID, err := strconv.ParseInt(problemIDs, 10, 64)
	if err != nil {
		logger.Log.Error("Error parsing the ID", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("I couldn't parse %s to an integer wtf", problemIDs)),
		)
		return
	}

	if err = c.Services.User.RemoveProblemFromClass(problemID); err != nil {
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
		return
	}

	logger.Log.Infof("Problem with ID: %d has been removed from this earthly plane and has passed to the astral plane", problemID)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf(
			"Problem with ID: %d has been removed from this earthly plane and has passed to the astral plane",
			problemID)),
	)
}

func checkIfUserIsMentor(c *context.ContextMessage) (bool, error) {
	// TODO: Remove this line after testing
	return true, nil

	// guildID, userID := c.Message.GuildID, c.Message.Author.ID
	// return c.DiscordValidator.IsMentor(guildID, userID)
}
