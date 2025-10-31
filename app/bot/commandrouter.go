package bot

import (
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/admin"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/enrolled"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/general"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/mentor"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
)

type CommandRouter func(c *context.ContextMessage)

var commandMap = map[string]CommandRouter{
	"!help":            routerHelp,
	"!bind":            routerBind,
	"!register":        routerRegister,
	"!deregister":      routerDeregister,
	"!list":            routerList,
	"!score":           routerScore,
	"!ac":              routerProposeAC,
	"!undo_ac":         routerUndoAC,
	"!add_problem":     routerAddProblem,
	"!remove_problem":  routerRemoveProblem,
	"!clear_class":     routerClearClass,
	"!create_class":    routerCreateClass,
	"!users":           routerGetUsersInClass,
	"!classes":         routerGetAllClasses,
	"!update_problem":  routerUpdateProblem,
	"!my_classes":      routerGetMyClasses,
	"!classes_of":      routerGetClassesOfUser,
	"!me":              routerGetMyUsername,
	"!change_username": routerChangeUsername,
	"!exterminate":     routerExterminate,
	"!export":          routerExport,
}

const NotEnough = "Not enough parameters bro"
const LimitExceed = "Parameters Limit Exceed"

func routerBind(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		fmt.Println(NotEnough)
	default:
		general.HandleBindUser(c)
	}
}

func routerRegister(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		fmt.Println(NotEnough)
	case 2:
		general.HandleRegisterUserInClass(c)
	case 3:
		mentor.HandleRegisterUser(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func routerDeregister(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		fmt.Println(NotEnough)
	case 2:
		enrolled.HandleDeregisterUser(c)
	case 3:
		mentor.HandleDeregisterUser(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func routerList(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		fmt.Println(NotEnough)
	case 2:
		general.HandleGetListOfProblemsByClass(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func routerScore(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		general.HandleGetScoreAllClasses(c)
	case 2:
		general.HandleGetScoreByClass(c)
	case 3:
		general.HandleGetScoreByUserAndClass(c)
	}
}

func routerProposeAC(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		fmt.Println(NotEnough)
	case 2:
		enrolled.HandleProposeAC(c)
	case 3:
		mentor.HandleAddAC(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func routerUndoAC(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		fmt.Println(NotEnough)
	case 2:
		enrolled.HandleUndoAC(c)
	case 3:
		mentor.HandleUndoAC(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func routerAddProblem(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1, 2, 3, 4:
		fmt.Println(NotEnough)
	default:
		mentor.HandleAddProblem(c)
	}
}

func routerRemoveProblem(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		fmt.Println(NotEnough)
	case 2:
		mentor.HandleRemoveProblem(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func routerClearClass(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		// Send confirmation message instead of directly executing
		classCode := c.Args[1]
		warningMessage := fmt.Sprintf("⚠️ **DANGER!** This will permanently delete class `%s` and all its data. This action cannot be undone!", classCode)

		c.Dispatcher.Send(
			dispatcher.MakeConfirmationMessage(c.Message.ChannelID, c.Message.ID, warningMessage),
		)
	default:
		fmt.Println(LimitExceed)
	}
}

func routerCreateClass(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1, 2:
		fmt.Println(NotEnough)
	default:
		admin.HandleCreateClass(c)
	}
}

func routerGetUsersInClass(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		fmt.Println(NotEnough)
	case 2:
		general.HandleGetUsersInClass(c)
	default:
		admin.HandleCreateClass(c)
	}
}

func routerGetAllClasses(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		general.HandleGetAllClasses(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func routerUpdateProblem(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1, 2, 3, 4, 5, 6:
		fmt.Println(NotEnough)
	default:
		mentor.HandleUpdateProblem(c)
	}
}

func routerGetMyClasses(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		general.HandleGetMyClasses(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func routerGetClassesOfUser(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		fmt.Println(NotEnough)
	case 2:
		general.HandleGetClassesOfUsername(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func routerGetMyUsername(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		general.HandleGetMyUsername(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func routerChangeUsername(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		fmt.Println(NotEnough)
	case 2:
		general.HandleChangeUsername(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func routerExterminate(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		fmt.Println(NotEnough)
	case 2:
		// Send confirmation message instead of executing directly
		warningMessage := fmt.Sprintf("⚠️ **DANGEROUS OPERATION**\n\n"+
			"This will **PERMANENTLY DELETE** all data for user `%s'\n"+
			"**This action CANNOT be undone!**", c.Args[1])

		c.Dispatcher.Send(dispatcher.MakeConfirmationMessage(
			c.Message.ChannelID,
			c.Message.ID,
			warningMessage,
		))
	default:
		fmt.Println(LimitExceed)
	}
}

func routerExport(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		fmt.Println(NotEnough)
	default:
		admin.HandleExportClass(c)
	}
}

// routerHelp handles the !help command, showing all commands or details for a specific command
func routerHelp(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		embed := buildGeneralHelpMessage()
		logger.Log.Debugf("Help embed built with %d fields", len(embed.Fields))
		c.Dispatcher.Send(
			dispatcher.MakeEmbedMessage(c.Message.ChannelID, embed),
		)
	case 2:
		command := c.Args[1]
		if info, exists := commandInfoMap["!"+command]; exists {
			embed := buildCommandHelpMessage(info)
			logger.Log.Debugf("Help embed built with %d fields", len(embed.Fields))
			c.Dispatcher.Send(
				dispatcher.MakeEmbedMessage(c.Message.ChannelID, embed),
			)
		} else {
			c.Dispatcher.Send(
				dispatcher.MakeSimpleMessage(c.Message.ChannelID, fmt.Sprintf("❌ Unknown command: `%s`\nUse `!help` to see all available commands.", command)),
			)
		}
	default:
		c.Dispatcher.Send(
			dispatcher.MakeSimpleMessage(c.Message.ChannelID, "Usage: `!help [command]`"),
		)
	}
}
