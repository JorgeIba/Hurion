package general

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/adapters"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/presentation/discord"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/presentation/image"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
	"github.com/bwmarrin/discordgo"
)

func HandleBindUser(c *context.ContextMessage) {
	userName := c.Args[1]
	discordID := c.Message.Author.ID
	discordNickName := c.Message.Author.GlobalName

	logger.Log.Infof("Binding %s to the user name %s...\n", discordNickName, userName)

	if err := c.Services.User.BindUser(service_model.User{UserID: discordID, UserName: userName}); err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionBindUser, userName)),
		)

		return
	}

	successMessage := fmt.Sprintf("User `%s` binded to the user name `%s`", discordNickName, userName)
	logger.Log.Info(successMessage)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, successMessage),
	)
}

func HandleRegisterUserInClass(c *context.ContextMessage) {
	classCode := c.Args[1]
	discordID, discordNickName := c.Message.Author.ID, c.Message.Author.GlobalName

	if err := c.Services.Enrollment.RegisterUserIDIntoClass(discordID, classCode); err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionRegisterInClass, classCode, discordNickName)),
		)

		return
	}

	successMessage := fmt.Sprintf("User `%s` has registered in the class `%s`", discordNickName, classCode)
	logger.Log.Info(successMessage)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, successMessage),
	)
}

func HandleGetListOfProblemsByClass(c *context.ContextMessage) {
	classCode := c.Args[1]

	groupedProblems, err := c.Services.Problem.GetProblemsGroupedByTopic(classCode)
	if err != nil {

		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionListProblemsByClass, classCode)),
		)

		return
	}

	if len(groupedProblems) == 0 {
		logger.Log.Infof("No problems found for class %s", classCode)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("No problems found for class `%s`", classCode)),
		)
		return
	}

	embeds := discord.FormatProblemsAsEmbeds(classCode, adapters.TransformToPresentationProblemsByTopic(groupedProblems))

	logger.Log.Info("Sending info")
	c.Dispatcher.Send(
		dispatcher.MessageResponse{
			Type:      dispatcher.PaginationEmbed,
			Session:   c.Session,
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
	userName, classCode := c.Args[1], c.Args[2]

	score, err := c.Services.Score.GetNeighborsScoreForUsername(userName, classCode)
	if err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionScoreByUserAndClass, userName, classCode)),
		)
		return
	}

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
		Title: fmt.Sprintf("Score of `%s` in the class `%s`", userName, classCode),
		Image: &discordgo.MessageEmbedImage{
			URL: "attachment://table.png",
		},
		Color: 0x00bfff,
	}

	logger.Log.Infof("Score of %s in class %s", userName, classCode)
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
		stringBuilder.WriteString(fmt.Sprintf("* <@%s>: %s\n", user.UserID, user.UserName))
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("List of users registered in class %s", classCode),
		Description: stringBuilder.String(),
		Color:       0x00ffcc,
	}

	logger.Log.Infof("Users in class %s\n%s", classCode, stringBuilder.String())
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

func HandleGetClassesOfUsername(c *context.ContextMessage) {
	userName := c.Args[1]

	userID, err := c.Services.User.GetUserIDByUsername(userName)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionGetClassesOfUsername, userName)),
		)
		return
	}

	classes, err := c.Services.User.GetClassesByUserID(userID)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
		)
		return
	}

	if len(classes) == 0 {
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Username `%s` not registered in any class", userName)),
		)
		return
	}

	var stringBuilder strings.Builder
	for _, class := range classes {
		stringBuilder.WriteString(fmt.Sprintf("* `%s`: %s\n", class.ClassCode, class.Name))
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Classes of user %s:", userName),
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

func HandleGetMyUsername(c *context.ContextMessage) {
	discordID := c.Message.Author.ID

	userName, err := c.Services.User.GetUsernameByUserID(discordID)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionGetMyUsername)),
		)
		return
	}

	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Your username is `%s`", userName)),
	)
}

func HandleChangeUsername(c *context.ContextMessage) {
	newUserName := c.Args[1]
	discordID := c.Message.Author.ID
	discordNickName := c.Message.Author.GlobalName

	// Check if user currently has a username bound
	hasUsernameBound, err := c.Services.User.HasUsernameBound(discordID)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
		)
		return
	}

	if err := c.Services.User.ChangeUsername(service_model.User{UserID: discordID, UserName: newUserName}); err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, adapters.MapError(err, adapters.ActionChangeUsername, newUserName)),
		)

		return
	}

	var successMessage string
	if !hasUsernameBound {
		successMessage = fmt.Sprintf("User `%s` bound to the username `%s` (no previous username found)", discordNickName, newUserName)
	} else {
		successMessage = fmt.Sprintf("User `%s` changed username to `%s`", discordNickName, newUserName)
	}

	logger.Log.Info(successMessage)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, successMessage),
	)
}
