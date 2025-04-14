package admin

import (
	"errors"
	"fmt"
	"strings"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
)

var (
	serviceConflictErr     = &serviceerr.ConflictError{}
	serviceNotFoundInfoErr = &serviceerr.NotFoundInfoError{}
)

func HandleClearClass(c *context.ContextMessage) {
	if isAdmin, err := checkIfUserIsAdmin(c); err != nil {
		logger.Log.Errorf("Error checking if author is admin: %v", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "There was a problem verifying permissions."),
		)
		return
	} else if !isAdmin {
		logger.Log.Infoln("Bro has not enough permissions (is not admin)")
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Bro has not enough permissions (is not admin)"),
		)
		return
	}

	classCode := c.Args[1]
	if err := c.Services.User.DeleteClassByClassCode(classCode); err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Class code `%s` not found", classCode)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
			)
		}

		return
	}

	logger.Log.Infof("Class %s has been removed\n", classCode)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Class `%s` has bee removed successfully", classCode)),
	)
}

func HandleCreateClass(c *context.ContextMessage) {
	if isAdmin, err := checkIfUserIsAdmin(c); err != nil {
		logger.Log.Errorf("Error checking if author is admin: %v\n", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "There was a problem verifying permissions."),
		)
		return
	} else if !isAdmin {
		logger.Log.Infoln("Bro has not enough permissions (is not admin)")
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Bro has not enough permissions (is not admin)"),
		)
		return
	}

	classCode, className := c.Args[1], strings.Join(c.Args[2:], " ")
	if err := c.Services.User.CreateClass(classCode, className); err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceConflictErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Class code `%s` already exists", classCode)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
			)
		}

		return
	}

	logger.Log.Infof("Class %s - %s created correctly!", classCode, className)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Class `%s - %s` created correctly!", classCode, className)),
	)
}

func checkIfUserIsAdmin(c *context.ContextMessage) (bool, error) {
	// TODO: Remove this line after testing
	return true, nil

	// guildID, userID := c.Message.GuildID, c.Message.Author.ID
	// return c.DiscordValidator.IsAdmin(guildID, userID)
}
