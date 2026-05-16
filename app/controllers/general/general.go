package general

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/adapters"
	controllers_common "github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/common"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/presentation/discord"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/presentation/image"
	"github.com/bwmarrin/discordgo"
)

func HandleGetListOfProblems(c *context.ContextMessage) {
	class, err := controllers_common.ResolveClassContext(c, 1)
	if err != nil {
		logger.Log.Error(err)

		switch {
		case serviceerr.IsServiceError(err):
			// It's a domain error, map it to a user-friendly message
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(controllers_common.GetMessageChannelID(c), adapters.MapError(err, adapters.ActionListProblemsByClass, controllers_common.ResolveClassCodeArg(c, 1))),
			)
		default:
			// controller error or unexpected error, send generic message
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(controllers_common.GetMessageChannelID(c), "Something wrong with the controller, bro. Contact the admin."),
			)
		}
		return
	}

	groupedProblems, err := c.Services.Problem.GetProblemsGroupedByTopic(class.ClassCode)
	if err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionListProblemsByClass, class.ClassCode)),
		)

		return
	}

	if len(groupedProblems) == 0 {
		logger.Log.Infof("No problems found for class %s", class.ClassCode)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("No problems found for class `%s`", class.ClassCode)),
		)
		return
	}

	embeds := discord.FormatProblemsAsEmbeds(class.ClassCode, adapters.TransformToPresentationProblemsByTopic(groupedProblems))

	logger.Log.Info("Sending info")
	c.Dispatcher.Send(
		dispatcher.MessageResponse{
			Type:      dispatcher.PaginationEmbed,
			ChannelID: c.Message.ChannelID,
			Embeds:    embeds,
		},
	)
}

func HandleGetScoreByClass(c *context.ContextMessage) {
	class, err := controllers_common.ResolveClassContext(c, 1)
	if err != nil {
		logger.Log.Error(err)

		switch {
		case serviceerr.IsServiceError(err):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(controllers_common.GetMessageChannelID(c), adapters.MapError(err, adapters.ActionScoreByClass, controllers_common.ResolveClassCodeArg(c, 1))),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(controllers_common.GetMessageChannelID(c), "Something wrong with the controller, bro. Contact the admin."),
			)
		}
		return
	}

	classCode := class.ClassCode

	title := fmt.Sprintf("Score of class `%s`", classCode)
	score, err := c.Services.Score.GetScoreByClassCode(classCode)
	if err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionScoreByClass, classCode)),
		)

		return
	}

	// Resolve display names
	controllers_common.ResolveScoreDisplayNames(c, c.ExecutorGuildID, &score)

	// Format score as table
	scoreTable := adapters.ScoreToTableScore(score)

	// Get Table as Image
	imgBytes, err := image.GenerateScoreImage(scoreTable)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
		)
		return
	}

	file := &discordgo.File{
		Name:   "table.png",
		Reader: bytes.NewReader(imgBytes),
	}

	embed := &discordgo.MessageEmbed{
		Title: title,
		Image: &discordgo.MessageEmbedImage{
			URL: "attachment://table.png",
		},
		Color: 0x00bfff,
	}

	logger.Log.Debug(title, scoreTable.Header, scoreTable.Table)
	c.Dispatcher.Send(dispatcher.MessageResponse{
		Type:      dispatcher.FileMessage,
		ChannelID: c.Message.ChannelID,
		Embeds:    []*discordgo.MessageEmbed{embed},
		Files:     []*discordgo.File{file},
	})
}

func HandleGetScoreByUserAndClass(c *context.ContextMessage) {
	userParam := c.Args[1]

	userID, ok := controllers_common.ParseUserID(userParam)
	if !ok {
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("bro `%s` is not a valid user mention or ID", userParam)),
		)
		return
	}

	class, err := controllers_common.ResolveClassContext(c, 2)
	if err != nil {
		logger.Log.Error(err)

		switch {
		case serviceerr.IsServiceError(err):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(controllers_common.GetMessageChannelID(c), adapters.MapError(err, adapters.ActionScoreByUserAndClass, userParam, controllers_common.ResolveClassCodeArg(c, 2))),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(controllers_common.GetMessageChannelID(c), "Something wrong with the controller, bro. Contact the admin."),
			)
		}
		return
	}

	classCode := class.ClassCode
	targetDisplayName := controllers_common.ResolveDisplayName(c, c.ExecutorGuildID, userID)

	score, err := c.Services.Score.GetNeighborsScoreForUserID(userID, classCode)
	if err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionScoreByUserAndClass, targetDisplayName, classCode)),
		)
		return
	}

	// Resolve display names
	controllers_common.ResolveScoreDisplayNames(c, c.ExecutorGuildID, &score)

	// Format score as table
	scoreTable := adapters.ScoreToTableScore(score)

	// Get Table as Image
	imgByte, err := image.GenerateScoreImage(scoreTable)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
		)
		return
	}

	file := &discordgo.File{
		Name:   "table.png",
		Reader: bytes.NewReader(imgByte),
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("Score of `%s` in the class `%s`", targetDisplayName, classCode),
		Image: &discordgo.MessageEmbedImage{
			URL: "attachment://table.png",
		},
		Color: 0x00bfff,
	}

	logger.Log.Debugf("Score of %s in class %s", userID, classCode)
	c.Dispatcher.Send(dispatcher.MessageResponse{
		Type:      dispatcher.FileMessage,
		ChannelID: c.Message.ChannelID,
		Embeds:    []*discordgo.MessageEmbed{embed},
		Files:     []*discordgo.File{file},
	})
}

func HandleGetUsersInClass(c *context.ContextMessage) {
	class, err := controllers_common.ResolveClassContext(c, 1)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionGetUsersInClass, "")),
		)
		return
	}
	classCode := class.ClassCode

	users, err := c.Services.Enrollment.GetListOfUsersByClassCode(classCode)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionGetUsersInClass, classCode)),
		)
		return
	}

	var stringBuilder strings.Builder
	for _, user := range users {
		displayName := controllers_common.ResolveDisplayName(c, c.ExecutorGuildID, user.UserID)
		stringBuilder.WriteString(fmt.Sprintf("* %s\n", displayName))
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("List of active users in class %s", classCode),
		Description: stringBuilder.String(),
		Color:       0x00ffcc,
	}

	logger.Log.Debugf("Users in class %s\n%s", classCode, stringBuilder.String())
	c.Dispatcher.Send(
		dispatcher.MessageResponse{
			Type:      dispatcher.EmbedMessage,
			ChannelID: c.Message.ChannelID,
			Embed:     embed,
		},
	)
}

func HandleGetScoreAllClasses(c *context.ContextMessage) {

	scoresMap, err := c.Services.Score.GetScoreAllClasses()
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected internal error."),
		)
		return
	}

	var embeds []*discordgo.MessageEmbed
	var files []*discordgo.File
	idx := 0
	for className, score := range scoresMap {
		// Resolve display names
		controllers_common.ResolveScoreDisplayNames(c, c.ExecutorGuildID, &score)

		// Format score to table
		scoreTable := adapters.ScoreToTableScore(score)

		// Get Table as Image
		imgBytes, err := image.GenerateScoreImage(scoreTable)
		if err != nil {
			logger.Log.Error(err)
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
			)
			return
		}

		fileName := fmt.Sprintf("table_%d.png", idx)

		file := &discordgo.File{
			Name:   fileName,
			Reader: bytes.NewReader(imgBytes),
		}

		embed := &discordgo.MessageEmbed{
			Title: fmt.Sprintf("Score of class %s", className),
			Image: &discordgo.MessageEmbedImage{
				URL: fmt.Sprintf("attachment://%s", fileName),
			},
			Color: 0x00bfff,
		}

		files = append(files, file)
		embeds = append(embeds, embed)
		logger.Log.Debug(className, scoreTable.Header, scoreTable.Table)
		idx++
	}

	c.Dispatcher.Send(dispatcher.MessageResponse{
		Type:      dispatcher.FileMessage,
		ChannelID: c.Message.ChannelID,
		Embeds:    embeds,
		Files:     files,
	})
}

func HandleGetAllClasses(c *context.ContextMessage) {
	classes, err := c.Services.Class.GetAllClasses()
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected internal error"),
		)
		return
	}

	var stringBuilder strings.Builder
	for _, class := range classes {
		stringBuilder.WriteString(fmt.Sprintf("* `%s`: %s\n", class.ClassCode, class.Name))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "All classes I found:",
		Description: stringBuilder.String(),
		Color:       0x00ffcc,
	}

	logger.Log.Info(stringBuilder.String())
	c.Dispatcher.Send(
		dispatcher.MessageResponse{
			Type:      dispatcher.EmbedMessage,
			ChannelID: c.Message.ChannelID,
			Embed:     embed,
		},
	)
}

func HandleGetMyClasses(c *context.ContextMessage) {
	principal, err := controllers_common.GetMemberSafe(c, c.Executor.UserID)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "I couldn't verify your roles with Discord, bro."),
		)
		return
	}

	classes, err := c.Services.User.GetAccessibleClassesByRoles(principal.Roles)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected internal error"),
		)
		return
	}

	if len(classes) == 0 {
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "You don't have access to any class roles, bro."),
		)
		return
	}

	var stringBuilder strings.Builder
	for _, class := range classes {
		stringBuilder.WriteString(fmt.Sprintf("* `%s`: %s\n", class.ClassCode, class.Name))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Your classes:",
		Description: stringBuilder.String(),
		Color:       0x00ffcc,
	}

	logger.Log.Info(stringBuilder.String())
	c.Dispatcher.Send(
		dispatcher.MessageResponse{
			Type:      dispatcher.EmbedMessage,
			ChannelID: c.Message.ChannelID,
			Embed:     embed,
		},
	)
}

func HandleGetClassesOfUser(c *context.ContextMessage) {
	userParam := c.Args[1]

	userID, ok := controllers_common.ParseUserID(userParam)
	if !ok {
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("bro `%s` is not a valid user mention or ID", userParam)),
		)
		return
	}

	principal, err := controllers_common.GetMemberSafe(c, userID)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "I couldn't verify that user's roles with Discord, bro."),
		)
		return
	}

	classes, err := c.Services.User.GetAccessibleClassesByRoles(principal.Roles)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionGetClassesOfUser, userParam)),
		)
		return
	}

	if len(classes) == 0 {
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("User <@%s> doesn't have access to any class roles, bro.", userID)),
		)
		return
	}

	var stringBuilder strings.Builder
	for _, class := range classes {
		stringBuilder.WriteString(fmt.Sprintf("* `%s`: %s\n", class.ClassCode, class.Name))
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Classes of user <@%s>:", userID),
		Description: stringBuilder.String(),
		Color:       0x00ffcc,
	}

	logger.Log.Info(stringBuilder.String())
	c.Dispatcher.Send(
		dispatcher.MessageResponse{
			Type:      dispatcher.EmbedMessage,
			ChannelID: c.Message.ChannelID,
			Embed:     embed,
		},
	)
}

func HandleGetClassRoles(c *context.ContextMessage) {
	class, err := controllers_common.ResolveClassContext(c, 1)
	if err != nil {
		logger.Log.Error(err)

		switch {
		case serviceerr.IsServiceError(err):
			// It's a domain error, map it to a user-friendly message
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(controllers_common.GetMessageChannelID(c), adapters.MapError(err, adapters.ActionGetClassRoles, controllers_common.ResolveClassCodeArg(c, 1))),
			)
		default:
			// controller error or unexpected error, send generic message
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(controllers_common.GetMessageChannelID(c), "Something wrong with the controller, bro. Contact the admin."),
			)
		}
		return
	}

	roles, err := c.Services.Class.GetAccessRoles(class.ClassCode)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionGetClassRoles, class.ClassCode)),
		)
		return
	}

	if len(roles) == 0 {
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("No roles linked to class `%s`", class.ClassCode)),
		)
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Roles linked to class `%s`:\n", class.ClassCode))
	for _, role := range roles {
		sb.WriteString(fmt.Sprintf("* <@&%s> (`%s`)\n", role.RoleID, role.RoleID))
	}

	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, sb.String()),
	)
}

func HandleGetClassRooms(c *context.ContextMessage) {
	class, err := controllers_common.ResolveClassContext(c, 1)
	if err != nil {
		logger.Log.Error(err)

		switch {
		case serviceerr.IsServiceError(err):
			// It's a domain error, map it to a user-friendly message
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(controllers_common.GetMessageChannelID(c), adapters.MapError(err, adapters.ActionGetClassRooms, controllers_common.ResolveClassCodeArg(c, 1))),
			)
		default:
			// controller error or unexpected error, send generic message
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(controllers_common.GetMessageChannelID(c), "Something wrong with the controller, bro. Contact the admin."),
			)
		}
		return
	}

	rooms, err := c.Services.Class.GetClassrooms(class.ClassCode)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionGetClassRooms, class.ClassCode)),
		)
		return
	}

	if len(rooms) == 0 {
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("No rooms linked to class `%s`", class.ClassCode)),
		)
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Rooms linked to class `%s`:\n", class.ClassCode))
	for _, room := range rooms {
		sb.WriteString(fmt.Sprintf("* <#%s> (`%s`)\n", room.RoomID, room.RoomID))
	}

	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, sb.String()),
	)
}
