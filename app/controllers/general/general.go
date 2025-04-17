package general

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/formatter"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
	"github.com/bwmarrin/discordgo"
)

var (
	serviceConflictErr     = &serviceerr.ConflictError{}
	serviceNotFoundInfoErr = &serviceerr.NotFoundInfoError{}
	serviceInternalErr     = &serviceerr.InternalError{}
)

func HandleBindUser(c *context.ContextMessage) {
	userName := c.Args[1]
	discordID := c.Message.Author.ID
	discordNickName := c.Message.Author.GlobalName

	logger.Log.Infof("Binding %s to the user name %s...\n", discordNickName, userName)

	if err := c.Services.User.BindUser(discordID, userName); err != nil {
		logger.Log.Error(err)

		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
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

	if err := c.Services.User.RegisterUserIDIntoClass(discordID, classCode); err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Class code `%s` not found or user `%s` not binded yet", classCode, discordNickName)),
			)
		case errors.As(err, &serviceConflictErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("User `%s` already registered on class `%s`", discordNickName, classCode)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
			)
		}

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

	problems, err := c.Services.User.GetListOfProblemsByClassCode(classCode)
	if err != nil {

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

	// Get Table
	title := fmt.Sprintf("List of problems of class `%s`", classCode)
	headers := []string{"ID", "Problem Name"}
	var rows [][]string
	for _, p := range problems {
		rows = append(rows, []string{fmt.Sprintf("%d", p.ID), p.Name})
	}

	// Get Table as Image
	imgByte, err := formatter.GenerateTableImage(headers, rows)
	if err != nil {
		logger.Log.Error(err)
	}

	file := &discordgo.File{
		Name:   "tabla.png",
		Reader: bytes.NewReader(imgByte),
	}

	embed := &discordgo.MessageEmbed{
		Title: title,
		Image: &discordgo.MessageEmbedImage{
			URL: "attachment://tabla.png",
		},
		Color: 0x00bfff,
	}

	logger.Log.Info(title, headers, rows)
	c.Dispatcher.Send(dispatcher.MessageResponse{
		Type:      dispatcher.FileMessage,
		ChannelID: c.Message.ChannelID,
		Embed:     embed,
		Files:     []*discordgo.File{file},
	})
}

func HandleGetScoreByClass(c *context.ContextMessage) {
	classCode := c.Args[1]

	title := fmt.Sprintf("Score of class `%s`", classCode)
	scoreTable, err := c.Services.User.GetSortedScoreByClassCodeAsTable(classCode)
	if err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Class code %s doesn't match with any class", classCode)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
			)
		}

		return
	}

	// Get Table as Image
	imgByte, err := formatter.GenerateTableImage(scoreTable.Header, scoreTable.Table)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
		)
		return
	}

	file := &discordgo.File{
		Name:   "tabla.png",
		Reader: bytes.NewReader(imgByte),
	}

	embed := &discordgo.MessageEmbed{
		Title: title,
		Image: &discordgo.MessageEmbedImage{
			URL: "attachment://tabla.png",
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

	scoreTable, err := c.Services.User.GetNeighborsScoreTableForUsername(userName, classCode)
	if err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Not found info for class code %s or userName %s", classCode, userName)),
			)
		case errors.As(err, &serviceInternalErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected internal error."),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("User %s is registered in %s? I didn't find any AC", userName, classCode)),
			)
		}
		return
	}

	// Get Table as Image
	imgByte, err := formatter.GenerateTableImage(scoreTable.Header, scoreTable.Table)
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
		)
		return
	}

	file := &discordgo.File{
		Name:   "tabla.png",
		Reader: bytes.NewReader(imgByte),
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("Score of `%s` in the class `%s`", userName, classCode),
		Image: &discordgo.MessageEmbedImage{
			URL: "attachment://tabla.png",
		},
		Color: 0x00bfff,
	}

	logger.Log.Infof("Score of %s in class %s\n%v\n%v", userName, classCode, scoreTable.Header, scoreTable.Table)
	c.Dispatcher.Send(dispatcher.MessageResponse{
		Type:      dispatcher.FileMessage,
		ChannelID: c.Message.ChannelID,
		Embeds:    []*discordgo.MessageEmbed{embed},
		Files:     []*discordgo.File{file},
	})
}

func HandleGetUsersInClass(c *context.ContextMessage) {
	classCode := c.Args[1]

	users, err := c.Services.User.GetUsersInClassCode(classCode)
	if err != nil {
		logger.Log.Error(err)
		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("No class was found with class code %s", classCode)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
			)
		}
	}

	var stringBuilder strings.Builder

	for _, user := range users {
		stringBuilder.WriteString(fmt.Sprintf("* <@%s>: %s\n", user.DiscordID, user.UserName))
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

	scoreTables, err := c.Services.User.GetScoreAllClasses()
	if err != nil {
		logger.Log.Error(err)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected internal error."),
		)
		return
	}

	var embeds []*discordgo.MessageEmbed
	var files []*discordgo.File
	for idx, scoreTable := range scoreTables {
		// Get Table as Image
		imgByte, err := formatter.GenerateTableImage(scoreTable.Header, scoreTable.Table)
		if err != nil {
			logger.Log.Error(err)
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
			)
			return
		}

		fileName := fmt.Sprintf("tabla_%d.png", idx)

		file := &discordgo.File{
			Name:   fileName,
			Reader: bytes.NewReader(imgByte),
		}

		embed := &discordgo.MessageEmbed{
			Title: fmt.Sprintf("Score of class %s", scoreTable.Class),
			Image: &discordgo.MessageEmbedImage{
				URL: fmt.Sprintf("attachment://%s", fileName),
			},
			Color: 0x00bfff,
		}

		files = append(files, file)
		embeds = append(embeds, embed)
		logger.Log.Debug(scoreTable.Class, scoreTable.Header, scoreTable.Table)
	}

	c.Dispatcher.Send(dispatcher.MessageResponse{
		Type:      dispatcher.FileMessage,
		ChannelID: c.Message.ChannelID,
		Embeds:    embeds,
		Files:     files,
	})
}

func HandleGetAllClasses(c *context.ContextMessage) {
	classes, err := c.Services.User.GetAllClasses()
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
