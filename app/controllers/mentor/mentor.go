package mentor

import (
	"strconv"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
)

func RegisterUserInClassByUser(c *context.ContextMessage) {
	if IsMentor, err := checkIfUserIsMentor(c); err != nil {
		logger.Log.Errorf("Error checking if author is admin: %v\n", err)
		return
	} else if !IsMentor {
		logger.Log.Errorf("Bro has not enough permissions (is not mentor)\n")
		return
	}

	userName, classCode := c.Args[1], c.Args[2]

	// Get class
	class, err := c.Datastore.GetClassByClassCode(classCode)
	className := class.Name
	if err != nil {
		logger.Log.Error(err)
		return
	}

	discordID, err := c.Datastore.GetDiscordIDByUserName(userName)
	if err != nil {
		logger.Log.Error(err)
		return
	}

	// Discord User should have binded a User Name
	isUserBinded, err := c.Datastore.IsUserBinded(discordID)
	if err != nil {
		logger.Log.Errorln(err)
		return
	} else if !isUserBinded {
		logger.Log.Infoln("User has not username binded yet")
		return
	}

	// Check if user was already registered
	isAlreadyRegistered, err := c.Datastore.IsUserRegisteredInClass(discordID, class.ID)
	if err != nil {
		logger.Log.Errorln(err)
		return
	} else if isAlreadyRegistered {
		logger.Log.Infoln("User has already registered to this class")
		return
	}

	c.Datastore.RegisterUserIntoClass(discordID, class.ID)
	logger.Log.Infof("User %s has registered in the class %s\n", userName, className)
}

func DeregisterUserByUserAndClass(c *context.ContextMessage) {
	if IsMentor, err := checkIfUserIsMentor(c); err != nil {
		logger.Log.Errorf("Error checking if author is admin: %v", err)
		return
	} else if !IsMentor {
		logger.Log.Infoln("Bro has not enough permissions (is not mentor)")
		return
	}

	userName, classCode := c.Args[1], c.Args[2]

	// Validate if user is registered in the class
	isMyBoyEnrolled, err := isUserEnrolledByClassCode(c, userName, classCode)
	if err != nil {
		logger.Log.Error(err)
		return
	} else if !isMyBoyEnrolled {
		logger.Log.Infof("User %s is not registered in class %s", userName, classCode)
		return
	}

	// Get Info
	classID, err := c.Datastore.GetClassIDByClassCode(classCode)
	if err != nil {
		logger.Log.Error(err)
		return
	}

	discordID, err := c.Datastore.GetDiscordIDByUserName(userName)
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

	logger.Log.Infof("User %s has registered to the class %s successfully", userName, classCode)
}

func UndoACByUser(c *context.ContextMessage) {
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

	discordID, err := c.Datastore.GetDiscordIDByUserName(userName)
	if err != nil {
		logger.Log.Error(err)
		return
	}

	// Validate problem exists
	if isProblemPresent, err := validateProblem(c, problemID); err != nil {
		logger.Log.Error(err)
		return
	} else if !isProblemPresent {
		logger.Log.Errorf("Are you sure problem %d exists?", problemID)
		return
	}

	// Validate user is registered in the class of the problem
	if isBroAbleToAC, err := broCanACProblem(c, discordID, problemID); err != nil {
		logger.Log.Error(err)
		return
	} else if !isBroAbleToAC {
		logger.Log.Infof("Bro %s cannot UndoAC the problem with id: %d", userName, problemID)
		return
	}

	// Undo-ing
	err = c.Datastore.RemoveAC(discordID, problemID)
	if err != nil {
		logger.Log.Error("Something happened", err)
		return
	}

	logger.Log.Infof("Problem %d has been UnAC-ed successfully by %s (or it wasn't even ACed previously, but I didn't want to check it tbh)", problemID, userName)
}

func AddProblem(c *context.ContextMessage) {
	if IsMentor, err := checkIfUserIsMentor(c); err != nil {
		logger.Log.Errorf("Error checking if author is admin: %v\n", err)
		return
	} else if !IsMentor {
		logger.Log.Errorln("Bro has not enough permissions (is not mentor)")
		return
	}

	problemName, classCode := c.Args[1], c.Args[2]

	if isClassCodeValid, err := validateClassCode(c, classCode); err != nil {
		logger.Log.Errorf("error checking if class code was valid: %v\n", err)
		return
	} else if !isClassCodeValid {
		logger.Log.Errorf("There's no class code: %v\n", err)
		return
	}

	classID, err := c.Datastore.GetClassIDByClassCode(classCode)
	if err != nil {
		logger.Log.Error(err)
		return
	}

	problemID, err := c.Datastore.AddProblem(problemName, classID)
	if err != nil {
		logger.Log.Error(err)
		return
	}

	logger.Log.Infof("Problem %s was added correctly to the class %s, Problem ID: %d\n", problemName, classCode, problemID)
}

func RemoveProblem(c *context.ContextMessage) {
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

	if isValidProblem, err := validateProblem(c, problemID); err != nil {
		logger.Log.Error(err)
		return
	} else if !isValidProblem {
		logger.Log.Errorf("Are you sure problem %d exists?", problemID)
		return
	}

	rows, err := c.Datastore.RemoveProblemByID(problemID)
	if err != nil {
		logger.Log.Error(err)
		return
	} else if rows == 0 {
		logger.Log.Infof("I think problem id was not existing, even tho it's already removed from this universe\n")
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

func isUserEnrolledByClassCode(c *context.ContextMessage, userID, classCode string) (bool, error) {
	return c.DatabaseValidator.IsEnrolledByClassCode(userID, classCode)
}

func validateProblem(c *context.ContextMessage, problemID int64) (bool, error) {
	return c.DatabaseValidator.ExistsProblem(problemID)
}

func validateClassCode(c *context.ContextMessage, classCode string) (bool, error) {
	return c.DatabaseValidator.IsClassCodeValid(classCode)
}

func broCanACProblem(c *context.ContextMessage, userID string, problemID int64) (bool, error) {
	problem, err := c.Datastore.GetProblemByID(problemID)
	if err != nil {
		return false, err
	}

	if isMyBoyEnrolled, err := c.DatabaseValidator.IsEnrolledByClassID(userID, problem.ClassID); err != nil {
		return false, err
	} else if !isMyBoyEnrolled {
		return false, nil
	}

	return true, nil
}
