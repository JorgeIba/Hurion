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

func HandleRegisterUserInClass(c *context.ContextMessage) {
	classCode := c.Args[1]
	user := c.Executor

	if err := c.Services.Enrollment.RegisterUserIntoClass(user, classCode); err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionRegisterInClass, classCode, user.DisplayName)),
		)

		return
	}

	successMessage := fmt.Sprintf("User `%s` has registered in the class `%s`", user.DisplayName, classCode)
	logger.Log.Info(successMessage)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, successMessage),
	)
}

func HandleGetListOfProblems(c *context.ContextMessage) {
	class, err := controllers_common.ResolveClassContext(c, 1)
	if err != nil {
		logger.Log.Error(err)

		switch {
		case serviceerr.IsServiceError(err):
			// It's a domain error, map it to a user-friendly message
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(controllers_common.GetMessageChannelID(c), adapters.MapError(err, adapters.ActionListProblemsByClass)),
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
	classCode := c.Args[1]

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
	classCode, userParam := c.Args[1], c.Args[2]

	userID, ok := controllers_common.ParseUserID(userParam)
	if !ok {
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("bro `%s` is not a valid user mention or ID", userParam)),
		)
		return
	}

	score, err := c.Services.Score.GetNeighborsScoreForUserID(userID, classCode)
	if err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionScoreByUserAndClass, userParam, classCode)),
		)
		return
	}

	// Resolve display names
	controllers_common.ResolveScoreDisplayNames(c, c.ExecutorGuildID, &score)

	// Resolve the target user's name for the title
	targetDisplayName := controllers_common.ResolveDisplayName(c, c.ExecutorGuildID, userID)

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
	classCode := c.Args[1]

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
		Title:       fmt.Sprintf("List of users registered in class %s", classCode),
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
	discordID := c.Message.Author.ID

	classes, err := c.Services.User.GetClassesByUserID(discordID)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected internal error"),
		)
		return
	}

	if len(classes) == 0 {
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "You are not registered in any class"),
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

	classes, err := c.Services.User.GetClassesByUserID(userID)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionGetClassesOfUser, userParam)),
		)
		return
	}

	if len(classes) == 0 {
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("User <@%s> not registered in any class", userID)),
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
