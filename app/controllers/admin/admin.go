package admin

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/JorgeIba/Hurion/app/context"
	"github.com/JorgeIba/Hurion/app/controllers/adapters"
	controllers_common "github.com/JorgeIba/Hurion/app/controllers/common"
	"github.com/JorgeIba/Hurion/app/customerr/controllerserr"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/presentation/excelizer"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/bwmarrin/discordgo"
)

func HandleClearClass(c *context.ContextMessage) error {

	class, err := controllers_common.ResolveClassContext(c, 1)
	if err != nil {
		logger.Log.Error(err)

		switch {
		case serviceerr.IsServiceError(err):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(controllers_common.GetMessageChannelID(c), adapters.MapError(err, adapters.ActionClearClass, controllers_common.ResolveClassCodeArg(c, 1))),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(controllers_common.GetMessageChannelID(c), "Something wrong with the controller, bro. Contact the admin."),
			)
		}

		return fmt.Errorf("error resolving class for clear_class: %w", err)
	}

	if err := c.Services.Class.ClearClassContentsByClassCode(services.UserActor(c.Executor.UserID), class.ClassCode); err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(controllers_common.GetMessageChannelID(c), adapters.MapError(err, adapters.ActionClearClass, class.ClassCode)),
		)

		return fmt.Errorf("error clearing class %s: %w", class.ClassCode, err)
	}

	logger.Log.Infof("Class %s has been cleared\n", class.ClassCode)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(controllers_common.GetMessageChannelID(c), fmt.Sprintf("Class `%s` has been cleared successfully", class.ClassCode)),
	)

	return nil
}

func HandleExterminateClass(c *context.ContextMessage) error {

	classCode := c.Args[1]
	if err := c.Services.Class.DeleteClassByClassCode(services.UserActor(c.Executor.UserID), classCode); err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionExterminateClass, classCode)),
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

	classCode, className := c.Args[1], strings.Join(c.Args[2:], " ")
	if !controllers_common.IsValidClassCode(classCode) {
		err := controllerserr.NewValidationError("class_code can only contain letters, numbers, underscores, and hyphens", nil)
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(controllers_common.GetMessageChannelID(c), "⚠️ Invalid class code! `class_code` can only contain letters, numbers, underscores, and hyphens (no slashes or special characters)."),
		)
		return
	}

	if _, err := c.Services.Class.CreateClass(services.UserActor(c.Executor.UserID), classCode, className); err != nil {
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

	mention := c.Args[1]
	userID, ok := controllers_common.ParseUserID(mention)
	if !ok {
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("bro `%s` is not a valid user mention or ID", mention)),
		)
		return nil
	}

	if err := c.Services.User.ExterminateUser(services.UserActor(c.Executor.UserID), userID); err != nil {
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

	classCode := c.Args[1]

	logger.Log.Infof("Generating Excel export for class %s", classCode)
	users, problemsByTopic, score, err := c.Services.Score.GetClassExportData(services.UserActor(c.Executor.UserID), classCode)
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

func HandleLinkRoles(c *context.ContextMessage) error {
	classCode := c.Args[1]
	rolesLinked := 0
	rolesAlreadyLinked := 0
	rolesFailed := 0
	var roleIDs []string
	for _, param := range c.Args[2:] {
		roleID, ok := controllers_common.ParseRoleID(param)
		if !ok {
			logger.Log.Warnf("Skipping invalid role param: %s", param)
			rolesFailed++
			continue
		}
		roleIDs = append(roleIDs, roleID)
	}

	results, err := c.Services.Class.LinkClassAccessRoles(services.UserActor(c.Executor.UserID), classCode, roleIDs...)
	if err != nil {
		logger.Log.Errorf("Failed to link roles to class %s: %v", classCode, err)
		c.Dispatcher.Send(dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionLinkRoles, classCode)))
		return err
	}

	for _, result := range results {
		if result.Err != nil {
			logger.Log.Errorf("Failed to link role %s to class %s: %v", result.ResourceID, classCode, result.Err)
			rolesFailed++
		} else if !result.Linked {
			rolesAlreadyLinked++
		} else {
			rolesLinked++
		}
	}

	msg := fmt.Sprintf("Successfully linked %d roles to class `%s`", rolesLinked, classCode)
	if rolesAlreadyLinked > 0 {
		msg += fmt.Sprintf(" (%d already linked)", rolesAlreadyLinked)
	}
	if rolesFailed > 0 {
		msg += fmt.Sprintf(" (%d failed/skipped)", rolesFailed)
	}
	c.Dispatcher.Send(dispatcher.MakeSimpleMessage(c.Message.ChannelID, msg))
	return nil
}

func HandleLinkRooms(c *context.ContextMessage) error {
	classCode := c.Args[1]
	roomsLinked := 0
	roomsAlreadyLinked := 0
	roomsFailed := 0
	var roomIDs []string
	for _, param := range c.Args[2:] {
		roomID, ok := controllers_common.ParseChannelID(param)
		if !ok {
			logger.Log.Warnf("Skipping invalid room param: %s", param)
			roomsFailed++
			continue
		}
		roomIDs = append(roomIDs, roomID)
	}

	results, err := c.Services.Class.LinkClassrooms(services.UserActor(c.Executor.UserID), classCode, roomIDs...)
	if err != nil {
		logger.Log.Errorf("Failed to link rooms to class %s: %v", classCode, err)
		c.Dispatcher.Send(dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionLinkRooms, classCode)))
		return err
	}

	for _, result := range results {
		if result.Err != nil {
			logger.Log.Errorf("Failed to link room %s to class %s: %v", result.ResourceID, classCode, result.Err)
			roomsFailed++
		} else if !result.Linked {
			roomsAlreadyLinked++
		} else {
			roomsLinked++
		}
	}

	msg := fmt.Sprintf("Successfully linked %d rooms to class `%s`", roomsLinked, classCode)
	if roomsAlreadyLinked > 0 {
		msg += fmt.Sprintf(" (%d already linked)", roomsAlreadyLinked)
	}
	if roomsFailed > 0 {
		msg += fmt.Sprintf(" (%d failed/skipped)", roomsFailed)
	}
	c.Dispatcher.Send(dispatcher.MakeSimpleMessage(c.Message.ChannelID, msg))
	return nil
}
