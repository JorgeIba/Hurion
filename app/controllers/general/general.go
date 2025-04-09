package general

import (
	"fmt"
	"strings"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
)

func BindUser(c *context.ContextMessage) {
	userName := c.Args[1]
	discordID := c.Message.Author.ID
	discordNickName := c.Message.Author.GlobalName

	logger.Log.Infof("Binding %s to the user name %s...\n", discordNickName, userName)

	err := c.Datastore.BindUser(discordID, userName)

	if err != nil {
		logger.Log.Infof("Some unexpected error happened: %v\n", err)
		return
	}

	logger.Log.Infof("User %s binded to the user name %s\n", discordNickName, userName)
}

func RegisterUserInClass(c *context.ContextMessage) {
	classCode := c.Args[1]

	// Get class
	class, err := c.Datastore.GetClassByClassCode(classCode)
	className := class.Name
	if err != nil {
		logger.Log.Error(err)
		return
	}

	discordID, discordNickName := c.Message.Author.ID, c.Message.Author.GlobalName

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
	logger.Log.Infof("User %s has registered in the class %s\n", discordNickName, className)
}

func GetListOfProblemsByClass(c *context.ContextMessage) {
	classCode := c.Args[1]

	classID, err := c.Datastore.GetClassIDByClassCode(classCode)
	if err != nil {
		logger.Log.Error(err)
		return
	}

	problems, err := c.Datastore.GetProblemsByClass(classID)
	if err != nil {
		logger.Log.Error(err)
		return
	}

	response := ""
	for _, problem := range problems {
		response += fmt.Sprintf("%d - %s\n", problem.ID, problem.Name)
	}
	logger.Log.Infoln(response)
}

func GetScoreByClass(c *context.ContextMessage) {
	classCode := c.Args[1]

	classID, err := c.Datastore.GetClassIDByClassCode(classCode)
	if err != nil {
		logger.Log.Error(err)
		return
	}

	studentsScore, err := c.Datastore.GetAllSolvedProblemsByClass(classID)
	if err != nil {
		logger.Log.Error(err)
		return
	}

	response := ""
	for user, solvedProblems := range studentsScore {
		problemIDs := make([]string, 0)
		for _, problem := range solvedProblems {
			problemIDs = append(problemIDs, fmt.Sprint(problem.ProblemID))
		}
		responseByUser := fmt.Sprintf("%s: [%v]", user.UserName, strings.Join(problemIDs, ", "))

		response += fmt.Sprintf("%v\n", responseByUser)
	}

	logger.Log.Info(response)
}

func GetScoreByUserAndClass(c *context.ContextMessage) {

}
