package mentor

import (
	"strconv"
	"strings"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/common"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
)

func HandleRegisterUser(c *context.ContextMessage) {
	if IsMentor, err := checkIfUserIsMentor(c); err != nil {
		logger.Log.Errorf("Error checking if author is admin: %v\n", err)
		return
	} else if !IsMentor {
		logger.Log.Errorf("Bro has not enough permissions (is not mentor)\n")
		return
	}

	userName, classCode := c.Args[1], c.Args[2]

	if err := c.Services.User.RegisterUsernameIntoClass(userName, classCode); err != nil {
		logger.Log.Error(err)
		return
	}

	logger.Log.Infof("User %s has registered in the class %s\n", userName, classCode)
}

func HandleDeregisterUser(c *context.ContextMessage) {
	if IsMentor, err := checkIfUserIsMentor(c); err != nil {
		logger.Log.Errorf("Error checking if author is admin: %v", err)
		return
	} else if !IsMentor {
		logger.Log.Infoln("Bro has not enough permissions (is not mentor)")
		return
	}

	userName, classCode := c.Args[1], c.Args[2]

	if err := c.Services.User.DeregisterUsernameFromClass(userName, classCode); err != nil {
		logger.Log.Error(err)
		return
	}

	logger.Log.Infof("User %s has registered to the class %s successfully", userName, classCode)
}

func HandlePendingConfirmation(c *context.ContextMessage) {
	if IsMentor, err := checkIfUserIsMentor(c); err != nil {
		logger.Log.Errorf("Error checking if author is admin: %v", err)
		return
	} else if !IsMentor {
		logger.Log.Infoln("Bro has not enough permissions (is not mentor)")
		return
	}

	messageID := c.MessageReaction.MessageID

	pendingConfirmation, err := c.Services.User.GetPendingConfirmationByMessageID(messageID)
	if err != nil {
		logger.Log.Infoln("MessageID not valid (or not found)")
		return
	}

	switch c.MessageReaction.Emoji.Name {
	case common.REACTION_NO:
		err := c.Services.User.RejectACProposal(messageID)
		if err != nil {
			logger.Log.Errorf("Something happened: %v", err)
			return
		}

		logger.Log.Info("Keep trying lil bro", pendingConfirmation)
	case common.REACTION_YES:
		err := c.Services.User.AcceptACProposal(messageID)
		if err != nil {
			logger.Log.Errorf("Something happened: %v", err)
			return
		}

		logger.Log.Infof("Good job bro, keep going. Problem %d has been AC-ed", pendingConfirmation.ProblemID)
	}
}

func HandleUndoAC(c *context.ContextMessage) {
	if IsMentor, err := checkIfUserIsMentor(c); err != nil {
		logger.Log.Errorf("Error checking if author is admin: %v", err)
		return
	} else if !IsMentor {
		logger.Log.Infoln("Bro has not enough permissions (is not mentor)")
		return
	}

	problemIDs, userName := c.Args[1], c.Args[2]
	problemID, err := strconv.ParseInt(problemIDs, 10, 64)
	if err != nil {
		logger.Log.Error("Error parsing the ID", err)
		return
	}

	if err = c.Services.User.UndoACByProblemAndUsername(problemID, userName); err != nil {
		logger.Log.Error(err)
		return
	}

	logger.Log.Infof("Problem %d has been UnAC-ed successfully by %s (or it wasn't even ACed previously, but I didn't want to check it tbh)", problemID, userName)
}

func HandleAddProblem(c *context.ContextMessage) {
	if IsMentor, err := checkIfUserIsMentor(c); err != nil {
		logger.Log.Errorf("Error checking if author is admin: %v\n", err)
		return
	} else if !IsMentor {
		logger.Log.Errorln("Bro has not enough permissions (is not mentor)")
		return
	}

	logger.Log.Info(c.Args)
	classCode, problemName := c.Args[1], strings.Join(c.Args[2:], " ")
	problemID, err := c.Services.User.CreateProblemIntoClass(problemName, classCode)
	if err != nil {
		logger.Log.Error(err)
		return
	}

	logger.Log.Infof("Problem %s was added correctly to the class %s, Problem ID: %d\n", problemName, classCode, problemID)
}

func HandleRemoveProblem(c *context.ContextMessage) {
	if IsMentor, err := checkIfUserIsMentor(c); err != nil {
		logger.Log.Errorf("Error checking if author is admin: %v\n", err)
		return
	} else if !IsMentor {
		logger.Log.Errorln("Bro has not enough permissions (is not mentor)")
		return
	}

	problemIDs := c.Args[1]
	problemID, err := strconv.ParseInt(problemIDs, 10, 64)
	if err != nil {
		logger.Log.Error("Error parsing the ID", err)
		return
	}

	if err = c.Services.User.RemoveProblemFromClass(problemID); err != nil {
		logger.Log.Error(err)
		return
	}

	logger.Log.Infof("Problem with ID: %d has been removed from this earthly plane and has passed to the astral plane", problemID)
}

func checkIfUserIsMentor(c *context.ContextMessage) (bool, error) {
	// TODO: Remove this line after testing
	return true, nil

	// guildID, userID := c.Message.GuildID, c.Message.Author.ID
	// return c.DiscordValidator.IsMentor(guildID, userID)
}
