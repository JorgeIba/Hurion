package general

import (
	"fmt"
	"strings"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
)

func HandleBindUser(c *context.ContextMessage) {
	userName := c.Args[1]
	discordID := c.Message.Author.ID
	discordNickName := c.Message.Author.GlobalName

	logger.Log.Infof("Binding %s to the user name %s...\n", discordNickName, userName)

	if err := c.Services.User.BindUser(discordID, userName); err != nil {
		logger.Log.Error(err)
		return
	}

	logger.Log.Infof("User %s binded to the user name %s\n", discordNickName, userName)
}

func HandleRegisterUserInClass(c *context.ContextMessage) {
	classCode := c.Args[1]
	discordID, discordNickName := c.Message.Author.ID, c.Message.Author.GlobalName

	if err := c.Services.User.RegisterUserIDIntoClass(discordID, classCode); err != nil {
		logger.Log.Error(err)
		return
	}

	logger.Log.Infof("User %s has registered in the class %s\n", discordNickName, classCode)
}

func HandleGetListOfProblemsByClass(c *context.ContextMessage) {
	classCode := c.Args[1]

	problems, err := c.Services.User.GetListOfProblemsByClassCode(classCode)
	if err != nil {
		logger.Log.Error(err)
		return
	}

	// Formatter
	response := ""
	for _, problem := range problems {
		response += fmt.Sprintf("%d - %s\n", problem.ID, problem.Name)
	}
	logger.Log.Infoln(response)
}

func HandleGetScoreByClass(c *context.ContextMessage) {
	classCode := c.Args[1]

	studentsScore, err := c.Services.User.GetScoreByClassCode(classCode)
	if err != nil {
		logger.Log.Error(err)
		return
	}

	// Formatter
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

func HandleGetScoreByUserAndClass(c *context.ContextMessage) {
	classCode := c.Args[1]

	studentsScore, err := c.Services.User.GetScoreByClassCode(classCode)
	if err != nil {
		logger.Log.Error(err)
		return
	}

	// Formatter
	// TODO: Only retrieve user and neighborhood
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
