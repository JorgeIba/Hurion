package general

import (
	"errors"
	"fmt"

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
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Class code `%s` not found", classCode)),
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
	headers := []any{"ID", "Problem Name"}
	var rows [][]any
	for _, p := range problems {
		rows = append(rows, []any{p.ID, p.Name})
	}

	// Markdown format
	mdTable := formatter.FormatToMarkdownTable("", headers, rows)

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("List of problems for class `%s`", classCode),
		Description: mdTable,
		Color:       0x00bfff,
	}

	c.Dispatcher.Send(dispatcher.MessageResponse{
		Type:      dispatcher.EmbedMessage,
		ChannelID: c.Message.ChannelID,
		Embed:     embed,
	})
}

func HandleGetScoreByClass(c *context.ContextMessage) {
	classCode := c.Args[1]

	score, err := c.Services.User.GetSortedScoreByClassCode(classCode)
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

	allProblems, err := c.Services.User.GetListOfProblemsByClassCode(classCode)
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

	headers := []any{"User"}
	for _, problem := range allProblems {
		headers = append(headers, problem.Name)
	}

	var rows [][]any
	for _, entry := range score {
		user := entry.User
		problemsOfUser := entry.Problems

		var row []any
		row = append(row, user.UserName)

		for _, problem := range allProblems {
			if v, exists := problemsOfUser[problem]; exists && v {
				row = append(row, "X")
			} else {
				row = append(row, " ")
			}
		}

		rows = append(rows, row)
	}

	mdTable := formatter.FormatToMarkdownTable("", headers, rows)

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Score of class `%s`", classCode),
		Description: mdTable,
		Color:       0x00bfff,
	}

	c.Dispatcher.Send(dispatcher.MessageResponse{
		Type:      dispatcher.EmbedMessage,
		ChannelID: c.Message.ChannelID,
		Embed:     embed,
	})
}

func HandleGetScoreByUserAndClass(c *context.ContextMessage) {
	userName, classCode := c.Args[1], c.Args[2]

	userID, err := c.Datastore.GetDiscordIDByUserName(userName)
	if err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("No userID is binded with userName `%s`", userName)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
			)
		}

		return
	}

	score, err := c.Services.User.GetSortedScoreByClassCode(classCode)
	if err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Class code %s not found", classCode)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
			)
		}

		return
	}

	allProblems, err := c.Services.User.GetListOfProblemsByClassCode(classCode)
	if err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Class code %s not found", classCode)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
			)
		}

		return
	}

	idxUserName := -1
	for idx, scoreUser := range score {
		if scoreUser.User.DiscordID == userID {
			idxUserName = idx
		}
	}

	if idxUserName == -1 {
		errMessage := fmt.Sprintf("User `%s` doesn't have any AC in class `%s`", userName, classCode)
		logger.Log.Error(errMessage)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, errMessage),
		)
		return
	}

	headers := []any{"User"}
	for _, problem := range allProblems {
		headers = append(headers, problem.Name)
	}

	var rows [][]any
	l, r := max(0, idxUserName-2), min(len(score)-1, idxUserName+2)
	for idx := l; idx <= r; idx++ {
		entry := score[idx]

		user := entry.User
		problemsOfUser := entry.Problems

		var row []any
		row = append(row, user.UserName)

		for _, problem := range allProblems {
			if v, exists := problemsOfUser[problem]; exists && v {
				row = append(row, "X")
			} else {
				row = append(row, " ")
			}
		}

		rows = append(rows, row)
	}

	title := fmt.Sprintf("Score of class `%s`", classCode)

	mdTable := formatter.FormatToMarkdownTable(title, headers, rows)

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Neighborhood of `%s`", userName),
		Description: mdTable,
		Color:       0x00bfff,
	}

	c.Dispatcher.Send(dispatcher.MessageResponse{
		Type:      dispatcher.EmbedMessage,
		ChannelID: c.Message.ChannelID,
		Embed:     embed,
	})
}
