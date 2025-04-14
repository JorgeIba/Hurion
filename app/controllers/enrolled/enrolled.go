package enrolled

import (
	"strconv"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
)

func HandleDeregisterUser(c *context.ContextMessage) {
	classCode := c.Args[1]
	discordID, discordNickname := c.Message.Author.ID, c.Message.Author.GlobalName

	// Validate if user is registered in the class
	isMyBoyEnrolled, err := isUserEnrolledInClassCode(c, classCode)
	if err != nil {
		logger.Log.Error(err)
		return
	} else if !isMyBoyEnrolled {
		logger.Log.Infof("User %s is not registered in class %s", discordNickname, classCode)
		return
	}

	if err := c.Services.User.DeregisterUserIDFromClass(discordID, classCode); err != nil {
		logger.Log.Error(err)
		return
	}

	logger.Log.Infof("User %s has registered to the class %s successfully", discordNickname, classCode)
}

func HandleProposeAC(c *context.ContextMessage) {
	problemIDs := c.Args[1]
	problemID, err := strconv.ParseInt(problemIDs, 10, 64)
	if err != nil {
		logger.Log.Error("Error parsing the ID", err)
		return
	}

	messageID, discordID, discordNickname := c.Message.ID, c.Message.Author.ID, c.Message.Author.GlobalName

	// Validate user is registered in the class of the problem
	if isBroAbleToAC, err := broCanACProblem(c, problemID); err != nil {
		logger.Log.Error(err)
		return
	} else if !isBroAbleToAC {
		logger.Log.Infof("Bro %s cannot AC the problem with id: %d", discordNickname, problemID)
		return
	}

	if err = c.Services.User.AddPendingACOfUserID(messageID, discordID, problemID); err != nil {
		logger.Log.Error(err)
		return
	}

	logger.Log.Info("Adding reactions...")
}

func HandleUndoAC(c *context.ContextMessage) {
	problemIDs := c.Args[1]
	problemID, err := strconv.ParseInt(problemIDs, 10, 64)
	if err != nil {
		logger.Log.Error("Error parsing the ID", err)
		return
	}

	discordID, discordNickname := c.Message.Author.ID, c.Message.Author.GlobalName

	// Validate user is registered in the class of the problem
	if isBroAbleToAC, err := broCanACProblem(c, problemID); err != nil {
		logger.Log.Error(err)
		return
	} else if !isBroAbleToAC {
		logger.Log.Infof("Bro %s cannot UndoAC the problem with id: %d", discordNickname, problemID)
		return
	}

	if err = c.Services.User.UndoACByProblemAndUserID(problemID, discordID); err != nil {
		logger.Log.Error(err)
		return
	}

	logger.Log.Infof("Problem %d has been UnAC-ed successfully by %s (or it wasn't even ACed previously, but I didn't want to check it tbh)", problemID, discordNickname)
}

func isUserEnrolledInClassCode(c *context.ContextMessage, classCode string) (bool, error) {
	userID := c.Message.Author.ID
	return c.DatabaseValidator.IsEnrolledByClassCode(userID, classCode)
}

func broCanACProblem(c *context.ContextMessage, problemID int64) (bool, error) {
	// TODO: Remove Datastore interaction in Controllers Layer
	problem, err := c.Datastore.GetProblemByID(problemID)
	if err != nil {
		return false, err
	}

	userID := c.Message.Author.ID
	if isMyBoyEnrolled, err := c.DatabaseValidator.IsEnrolledByClassID(userID, problem.ClassID); err != nil {
		return false, err
	} else if !isMyBoyEnrolled {
		return false, nil
	}

	return true, nil
}
