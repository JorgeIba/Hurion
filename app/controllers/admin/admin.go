package admin

import (
	"strings"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
)

func ClearClass(c *context.ContextMessage) {
	if isAdmin, err := checkIfAdmin(c); err != nil {
		logger.Log.Errorf("Error checking if author is admin: %v", err)
		return
	} else if !isAdmin {
		logger.Log.Infoln("Bro has not enough permissions (is not admin)")
		return
	}

	classCode := c.Args[1]

	classID, err := c.Datastore.GetClassIDByClassCode(classCode)
	if err != nil {
		logger.Log.Errorf("Error retrieving the classID: %v", err)
		return
	}

	if err = c.Datastore.ClearClass(classID); err != nil {
		logger.Log.Errorf("Error deleting the class: %v", err)
		return
	}

	logger.Log.Infoln("Class has been removed")
}

func CreateClass(c *context.ContextMessage) {
	if isAdmin, err := checkIfAdmin(c); err != nil {
		logger.Log.Errorf("Error checking if author is admin: %v\n", err)
		return
	} else if !isAdmin {
		logger.Log.Infoln("Bro has not enough permissions (is not admin)")
		return
	}

	classCode, className := c.Args[1], strings.Join(c.Args[2:], " ")
	_, err := c.Datastore.CreateClass(classCode, className)
	if err != nil {
		logger.Log.Errorf("Error creating the class: %v\n", err)
		return
	}

	logger.Log.Infoln("Class created correctly!")
}

func checkIfAdmin(c *context.ContextMessage) (bool, error) {
	// TODO: Remove this line after testing
	return true, nil

	// guildID, userID := c.Message.GuildID, c.Message.Author.ID
	// return c.DiscordValidator.IsAdmin(guildID, userID)
}
