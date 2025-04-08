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
	classCode := c.Args[1]

	// Get classID and className
	classID, err := c.Datastore.GetClassIDByClassCode(classCode)
	if err != nil {
		logger.Log.Error(err)
		return
	}

	className, err := c.Datastore.GetClassNameByClassID(classID)
	if err != nil {
		logger.Log.Error(err)
	}

	discordID := c.Message.Author.ID
	discordNickName := c.Message.Author.GlobalName

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
	isAlreadyRegistered, err := c.Datastore.IsUserRegisteredInClass(discordID, classID)
	if err != nil {
		logger.Log.Errorln(err)
		return
	} else if isAlreadyRegistered {
		logger.Log.Infoln("User has already registered to this class")
		return
	}

	c.Datastore.RegisterUserIntoClass(discordID, classID)
	logger.Log.Infof("User %s has registered in the class %s\n", discordNickName, className)
}

func GetListOfProblemsByClass(c *context.ContextMessage) {

}

func GetScoreByClass(c *context.ContextMessage) {

}

func GetScoreByUserAndClass(c *context.ContextMessage) {

}
