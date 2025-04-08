package enrolled

import (
	"strconv"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
)

func DeregisterUser(c *context.ContextMessage) {
	classCode := c.Args[1]
	discordID, discordNickname := c.Message.Author.ID, c.Message.Author.GlobalName

	// Validate if user is registered in the class
	isMyBoyEnrolled, err := isEnrolledByClassCode(c, classCode)
	if err != nil {
		logger.Log.Error(err)
		return
	} else if !isMyBoyEnrolled {
		logger.Log.Infof("User %s is not registered in class %s", discordNickname, classCode)
		return
	}

	// Get Info
	classID, err := c.Datastore.GetClassIDByClassCode(classCode)
	if err != nil {
		logger.Log.Error(err)
		return
	}

	// Deregister
	err = c.Datastore.DeregisterUserFromClass(discordID, classID)
	if err != nil {
		logger.Log.Error(err)
		return
	}

	logger.Log.Infof("User %s has registered to the class %s successfully", discordNickname, classCode)
}

func AddAC(c *context.ContextMessage) {
	problemIDs := c.Args[1]
	problemID, err := strconv.ParseInt(problemIDs, 10, 64)
	if err != nil {
		logger.Log.Error("Error parsing the ID", err)
		return
	}

	discordID, discordNickname := c.Message.Author.ID, c.Message.Author.GlobalName

	// Validate problem exists
	if isProblemPresent, err := existsProblem(c, problemID); err != nil {
		logger.Log.Error(err)
		return
	} else if !isProblemPresent {
		logger.Log.Errorf("Are you sure problem %d exists?", problemID)
		return
	}

	// Validate user is registered in the class of the problem
	if isBroAbleToAC, err := broCanACProblem(c, problemID); err != nil {
		logger.Log.Error(err)
		return
	} else if !isBroAbleToAC {
		logger.Log.Infof("Bro %s cannot AC the problem with id: %d", discordNickname, problemID)
		return
	}

	// ACing
	err = c.Datastore.AddAC(discordID, problemID)
	if err != nil {
		logger.Log.Error("Something happened", err)
		return
	}

	logger.Log.Infof("Good job bro, keep going. Problem %d has been AC-ed successfully by %s", problemID, discordNickname)
}

func UndoAC(c *context.ContextMessage) {
	problemIDs := c.Args[1]
	problemID, err := strconv.ParseInt(problemIDs, 10, 64)
	if err != nil {
		logger.Log.Error("Error parsing the ID", err)
		return
	}

	discordID, discordNickname := c.Message.Author.ID, c.Message.Author.GlobalName

	// Validate problem exists
	if isProblemPresent, err := existsProblem(c, problemID); err != nil {
		logger.Log.Error(err)
		return
	} else if !isProblemPresent {
		logger.Log.Errorf("Are you sure problem %d exists?", problemID)
		return
	}

	// Validate user is registered in the class of the problem
	if isBroAbleToAC, err := broCanACProblem(c, problemID); err != nil {
		logger.Log.Error(err)
		return
	} else if !isBroAbleToAC {
		logger.Log.Infof("Bro %s cannot UndoAC the problem with id: %d", discordNickname, problemID)
		return
	}

	// Undo-ing
	err = c.Datastore.RemoveAC(discordID, problemID)
	if err != nil {
		logger.Log.Error("Something happened", err)
		return
	}

	logger.Log.Infof("Problem %d has been UnAC-ed successfully by %s (or it wasn't even ACed previously, but I didn't want to check it tbh)", problemID, discordNickname)
}

func isEnrolledByClassCode(c *context.ContextMessage, classCode string) (bool, error) {
	userID := c.Message.Author.ID
	return c.DatabaseValidator.IsEnrolledByClassCode(userID, classCode)
}

func existsProblem(c *context.ContextMessage, problemID int64) (bool, error) {
	return c.DatabaseValidator.ExistsProblem(problemID)
}

func broCanACProblem(c *context.ContextMessage, problemID int64) (bool, error) {
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
