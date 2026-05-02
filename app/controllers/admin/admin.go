package admin

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/adapters"
	controllers_common "github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/common"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/presentation/excelizer"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
	"github.com/bwmarrin/discordgo"
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
		return nil
	}

	classCode := c.Args[1]
	if err := c.Services.Class.DeleteClassByClassCode(classCode); err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionClearClass, classCode)),
		)

		return fmt.Errorf("error deleting class %s: %w", classCode, err)
	}

	logger.Log.Infof("Class %s has been removed\n", classCode)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Class `%s` has been removed successfully", classCode)),
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
	if _, err := c.Services.Class.CreateClass(classCode, className); err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionCreateClass, classCode)),
		)

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
		return nil
	}

	mention := c.Args[1]
	userID, ok := controllers_common.ParseUserID(mention)
	if !ok {
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("bro `%s` is not a valid user mention or ID", mention)),
		)
		return nil
	}

	if err := c.Services.User.ExterminateUser(userID); err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionExterminateUser, userID)),
		)

		return fmt.Errorf("error exterminating user %s: %w", userID, err)
	}

	logger.Log.Infof("User %s exterminated successfully!", userID)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("User <@%s> has been completely removed from the system.", userID)),
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

	classCode := c.Args[1]

	logger.Log.Infof("Generating Excel export for class %s", classCode)
	users, problemsByTopic, score, err := c.Services.Score.GetClassExportData(classCode)
	if err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionExportClass, classCode)),
		)

		return fmt.Errorf("error getting data for class %s: %w", classCode, err)
	}

	// Resolve display names
	controllers_common.ResolveUsersDisplayNames(c, c.ExecutorGuildID, users)
	controllers_common.ResolveScoreDisplayNames(c, c.ExecutorGuildID, &score)

	excelizerUsers := adapters.TransformToExcelizerUsers(users)
	excelizerProblemsByTopic := adapters.TransformToExcelizerProblemsByTopic(problemsByTopic)
	tableScore := adapters.ScoreToTableScore(score)
	// Generate Excel file and send it as a Discord message
	excelBytes, filename, err := excelizer.ExportClassToExcel(excelizerUsers, excelizerProblemsByTopic, tableScore, classCode)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error while generating Excel file."),
		)
		return fmt.Errorf("error generating excel for class %s: %w", classCode, err)
	}

	file := &discordgo.File{
		Name:        filename,
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Reader:      bytes.NewReader(excelBytes),
	}

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
	if c.Executor.UserID == "" {
		return false, fmt.Errorf("no executor user ID available")
	}

	if c.ExecutorGuildID == "" {
		return false, fmt.Errorf("admin commands can only be used in a server, not in DMs")
	}

	return c.DiscordValidator.IsAdmin(c.ExecutorGuildID, c.Executor.UserID)
}

func HandleLinkRoles(c *context.ContextMessage) error {
	if isAdmin, err := checkIfUserIsAdmin(c); err != nil {
		logger.Log.Errorf("Error checking admin status: %v", err)
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

	classCode := c.Args[1]
	rolesLinked := 0
	rolesFailed := 0
	for i := 2; i < len(c.Args); i++ {
		roleParam := c.Args[i]
		roleID, ok := controllers_common.ParseRoleID(roleParam)
		if !ok {
			logger.Log.Warnf("Skipping invalid role param: %s", roleParam)
			rolesFailed++
			continue
		}

		err := c.Services.Class.AssignAccessRole(service_model.ClassAccessRole{ClassCode: classCode, RoleID: roleID})
		if err != nil {
			logger.Log.Errorf("Failed to link role %s to class %s: %v", roleID, classCode, err)
			rolesFailed++
		} else {
			rolesLinked++
		}
	}

	msg := fmt.Sprintf("Successfully linked %d roles to class `%s`", rolesLinked, classCode)
	if rolesFailed > 0 {
		msg += fmt.Sprintf(" (%d failed/skipped)", rolesFailed)
	}
	c.Dispatcher.Send(dispatcher.MakeSimpleMessage(c.Message.ChannelID, msg))
	return nil
}

func HandleLinkRooms(c *context.ContextMessage) error {
	if isAdmin, err := checkIfUserIsAdmin(c); err != nil {
		logger.Log.Errorf("Error checking admin status: %v", err)
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

	classCode := c.Args[1]
	roomsLinked := 0
	roomsFailed := 0
	for i := 2; i < len(c.Args); i++ {
		roomParam := c.Args[i]
		roomID, ok := controllers_common.ParseChannelID(roomParam)
		if !ok {
			logger.Log.Warnf("Skipping invalid room param: %s", roomParam)
			roomsFailed++
			continue
		}

		err := c.Services.Class.RegisterClassroom(service_model.Classroom{RoomID: roomID, ClassCode: classCode})
		if err != nil {
			logger.Log.Errorf("Failed to link room %s to class %s: %v", roomID, classCode, err)
			roomsFailed++
		} else {
			roomsLinked++
		}
	}

	msg := fmt.Sprintf("Successfully linked %d rooms to class `%s`", roomsLinked, classCode)
	if roomsFailed > 0 {
		msg += fmt.Sprintf(" (%d failed/skipped)", roomsFailed)
	}
	c.Dispatcher.Send(dispatcher.MakeSimpleMessage(c.Message.ChannelID, msg))
	return nil
}
