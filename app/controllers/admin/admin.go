package admin

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
	"github.com/bwmarrin/discordgo"
)

var (
	serviceConflictErr     = &serviceerr.ConflictError{}
	serviceNotFoundInfoErr = &serviceerr.NotFoundInfoError{}
)

func HandleClearClass(c *context.ContextMessage) error {
	if isAdmin, err := checkIfUserIsAdmin(c); err != nil {
		logger.Log.Errorf("Error checking if author is admin: %v", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "There was a problem verifying permissions."),
		)
		return err
	} else if !isAdmin {
		logger.Log.Infoln("Bro has not enough permissions (is not admin)")
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Bro has not enough permissions (is not admin)"),
		)
		return err
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

		return fmt.Errorf("error deleting class %s: %w", classCode, err)
	}

	logger.Log.Infof("Class %s has been removed\n", classCode)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Class `%s` has bee removed successfully", classCode)),
	)

	return nil
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

func HandleExterminate(c *context.ContextMessage) error {
	if isAdmin, err := checkIfUserIsAdmin(c); err != nil {
		logger.Log.Errorf("Error checking if author is admin: %v", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "There was a problem verifying permissions."),
		)
		return err
	} else if !isAdmin {
		logger.Log.Infoln("Bro has not enough permissions (is not admin)")
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Bro has not enough permissions (is not admin)"),
		)
		return err
	}

	userID := c.Args[1]
	if err := c.Services.User.ExterminateUser(userID); err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("User `%s` not found in the system", userID)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error while exterminating user."),
			)
		}

		return fmt.Errorf("error exterminating user %s: %w", userID, err)
	}

	logger.Log.Infof("User %s exterminated successfully!", userID)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("User `%s` has been completely removed from the system.", userID)),
	)

	return nil
}

func HandleExportClass(c *context.ContextMessage) error {
	if isAdmin, err := checkIfUserIsAdmin(c); err != nil {
		logger.Log.Errorf("Error checking if author is admin: %v", err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "There was a problem verifying permissions."),
		)
		return err
	} else if !isAdmin {
		logger.Log.Infoln("Bro has not enough permissions (is not admin)")
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Bro has not enough permissions (is not admin)"),
		)
		return nil
	}

	// Extract class_code from args
	classCode := c.Args[1]

	// Call service to generate Excel file
	excelBytes, filename, err := c.Services.User.ExportClassToExcel(classCode)
	if err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Class code `%s` not found", classCode)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error while exporting class data."),
			)
		}

		return fmt.Errorf("error exporting class %s: %w", classCode, err)
	}

	// Create Discord file attachment
	file := &discordgo.File{
		Name:        filename,
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Reader:      bytes.NewReader(excelBytes),
	}

	// Send file via dispatcher
	logger.Log.Infof("Sending Excel export for class %s (`%s`)", classCode, filename)
	c.Dispatcher.Send(
		dispatcher.MakeFileMessage(
			c.Message.ChannelID,
			fmt.Sprintf("Excel export for class `%s`", classCode),
			[]*discordgo.File{file},
		),
	)

	return nil
}

func checkIfUserIsAdmin(c *context.ContextMessage) (bool, error) {
	// TODO: Remove this line after testing
	return true, nil

	// guildID, userID := c.Message.GuildID, c.Message.Author.ID
	// return c.DiscordValidator.IsAdmin(guildID, userID)
}
