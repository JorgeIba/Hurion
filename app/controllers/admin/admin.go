package admin

import (
	"strings"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
)

func HandleClearClass(c *context.ContextMessage) {
	if isAdmin, err := checkIfUserIsAdmin(c); err != nil {
		logger.Log.Errorf("Error checking if author is admin: %v", err)
		return
	} else if !isAdmin {
		logger.Log.Infoln("Bro has not enough permissions (is not admin)")
		return
	}

	classCode := c.Args[1]
	if err := c.Services.User.DeleteClassByClassCode(classCode); err != nil {
		logger.Log.Error(err)
		return
	}

	logger.Log.Infoln("Class has been removed")
}

func HandleCreateClass(c *context.ContextMessage) {
	if isAdmin, err := checkIfUserIsAdmin(c); err != nil {
		logger.Log.Errorf("Error checking if author is admin: %v\n", err)
		return
	} else if !isAdmin {
		logger.Log.Infoln("Bro has not enough permissions (is not admin)")
		return
	}

	classCode, className := c.Args[1], strings.Join(c.Args[2:], " ")
	if err := c.Services.User.CreateClass(classCode, className); err != nil {
		logger.Log.Error(err)
		return
	}

	logger.Log.Infoln("Class created correctly!")
}

func checkIfUserIsAdmin(c *context.ContextMessage) (bool, error) {
	// TODO: Remove this line after testing
	return true, nil

	// guildID, userID := c.Message.GuildID, c.Message.Author.ID
	// return c.DiscordValidator.IsAdmin(guildID, userID)
}
