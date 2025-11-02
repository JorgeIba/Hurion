package admin

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/adapters"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/presentation/excelizer"
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
	if err := c.Services.Class.DeleteClassByClassCode(classCode); err != nil {
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
	if err := c.Services.Class.CreateClass(classCode, className); err != nil {
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

	// Get data from service (users, problems by topic, and score)
	users, problemsByTopic, score, err := c.Services.Score.GetClassExportData(classCode)
	if err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Class code `%s` not found", classCode)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error while retrieving class data."),
			)
		}

		return fmt.Errorf("error getting data for class %s: %w", classCode, err)
	}

	// Transform to presentation layer types
	excelizerUsers := adapters.TransformToExcelizerUsers(users)
	excelizerProblemsByTopic := adapters.TransformToExcelizerProblemsByTopic(problemsByTopic)
	tableScore := adapters.ScoreToTableScore(score)

	// Generate Excel file
	excelBytes, filename, err := excelizer.ExportClassToExcel(excelizerUsers, excelizerProblemsByTopic, tableScore, classCode)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error while generating Excel file."),
		)
		return fmt.Errorf("error generating excel for class %s: %w", classCode, err)
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
	if c.ExecutorUserID == "" {
		return false, fmt.Errorf("no executor user ID available")
	}

	if c.ExecutorGuildID == "" {
		return false, fmt.Errorf("admin commands can only be used in a server, not in DMs")
	}

	return c.DiscordValidator.IsAdmin(c.ExecutorGuildID, c.ExecutorUserID)
}
