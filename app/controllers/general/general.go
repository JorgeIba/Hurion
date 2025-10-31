package general

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/formatter"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services"
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

		switch {
		case errors.As(err, &serviceConflictErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Username `%s` is already taken by another user. Please choose a different username.", userName)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
			)
		}

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

	groupedProblems, err := c.Services.User.GetProblemsGroupedByTopic(classCode)
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

	if len(groupedProblems) == 0 {
		logger.Log.Infof("No problems found for class %s", classCode)
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("No problems found for class `%s`", classCode)),
		)
		return
	}

	var embeds []*discordgo.MessageEmbed
	title := fmt.Sprintf("List of problems for class %s", classCode)

	// Build content by iterating through grouped problems
	body := ""
	problemCount := 0
	const LENGTH_PAGE = 5

	for _, group := range groupedProblems {
		// Add topic header
		if body != "" {
			body += "\n"
		}
		body += fmt.Sprintf("**%s**\n", group.Topic)

		// Add problems in this topic
		for _, problem := range group.Problems {
			difficultyEmoji := getDifficultyEmoji(problem.GetDifficulty())
			body += fmt.Sprintf("%d. %s - [%s](%s)\n", problem.ID, difficultyEmoji, problem.Name, problem.URL)
			problemCount++

			// Create a new page every LENGTH_PAGE problems
			if problemCount%LENGTH_PAGE == 0 {
				embeds = append(embeds, &discordgo.MessageEmbed{
					Title:       title,
					Description: body,
				})
				body = ""
			}
		}
	}

	// Add any remaining content as the last page
	if body != "" {
		embeds = append(embeds, &discordgo.MessageEmbed{
			Title:       title,
			Description: body,
		})
	}

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
	score, err := c.Services.User.GetScoreByClassCode(classCode)
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

	// Format score as table
	scoreTable := services.ScoreToTableScore(score)

	// Get Table as Image
	imgByte, err := formatter.GenerateTableImage(scoreTable)
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

	score, err := c.Services.User.GetNeighborsScoreForUsername(userName, classCode)
	if err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Not found info for class code `%s` or user name `%s`", classCode, userName)),
			)
		case errors.As(err, &serviceInternalErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected internal error."),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("User `%s` is registered in `%s`? I didn't find any AC", userName, classCode)),
			)
		}
		return
	}

	// Format score as table
	scoreTable := services.ScoreToTableScore(score)

	// Get Table as Image
	imgByte, err := formatter.GenerateTableImage(scoreTable)
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

	users, err := c.Services.User.GetListOfUsersByClassCode(classCode)
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

	scoresMap, err := c.Services.User.GetScoreAllClasses()
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
		scoreTable := services.ScoreToTableScore(score)
		
		// Get Table as Image
		imgByte, err := formatter.GenerateTableImage(scoreTable)
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
		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Username `%s` not found", userName)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
			)
		}
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
		switch {
		case errors.As(err, &serviceNotFoundInfoErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "You are not binded to any username. Use `!bind <username>` to bind your account."),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
			)
		}
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

	if err := c.Services.User.ChangeUsername(discordID, newUserName); err != nil {
		logger.Log.Error(err)

		switch {
		case errors.As(err, &serviceConflictErr):
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("Username `%s` is already taken by another user. Please choose a different username.", newUserName)),
			)
		default:
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Unexpected error."),
			)
		}

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

func getDifficultyEmoji(difficulty data.Difficulty) string {
	switch difficulty {
	case data.DifficultyEasy:
		return "🟩"
	case data.DifficultyMedium:
		return "🟨"
	case data.DifficultyHard:
		return "🟥"
	default:
		return "⬜"
	}
}
