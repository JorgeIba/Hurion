package general

import (
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

}

func GetListOfProblemsByClass(c *context.ContextMessage) {

}

func GetScoreByClass(c *context.ContextMessage) {

}

func GetScoreByUserAndClass(c *context.ContextMessage) {

}
