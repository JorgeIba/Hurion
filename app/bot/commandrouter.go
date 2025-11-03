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

func sendUsageError(c *context.ContextMessage, command string) {
	message := fmt.Sprintf("Mmmm Invalid usage bro. Check `!help %s` for more info.", command)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, message),
	)
}

func routerBind(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		general.HandleBindUser(c)
	default:
		sendUsageError(c, "bind")
	}
}

func routerRegister(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		general.HandleRegisterUserInClass(c)
	case 3:
		mentor.HandleRegisterUser(c)
	default:
		sendUsageError(c, "register")
	}
}

func routerDeregister(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2, 3:
		// This command requires confirmation
		// confirmationHandler.go will handle it
		c.Dispatcher.Send(dispatcher.MakeConfirmationMessage(
			c.Message.ChannelID,
			c.Message.ID,
			GetConfirmationMessage("!deregister", c.Args),
		))
	default:
		sendUsageError(c, "deregister")
	}
}

func routerList(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		general.HandleGetListOfProblemsByClass(c)
	default:
		sendUsageError(c, "list")
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
	case 2:
		enrolled.HandleProposeAC(c)
	case 3:
		mentor.HandleAddAC(c)
	default:
		sendUsageError(c, "ac")
	}
}

func routerUndoAC(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		enrolled.HandleUndoAC(c)
	case 3:
		mentor.HandleUndoAC(c)
	default:
		sendUsageError(c, "undo_ac")
	}
}

func routerAddProblem(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1, 2, 3, 4:
		fmt.Println(NotEnough)
		sendUsageError(c, "add_problem")
	default:
		mentor.HandleAddProblem(c)
	}
}

func routerRemoveProblem(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		// This command requires confirmation
		// confirmationHandler.go will handle it
		c.Dispatcher.Send(dispatcher.MakeConfirmationMessage(
			c.Message.ChannelID,
			c.Message.ID,
			GetConfirmationMessage("!remove_problem", c.Args),
		))
	default:
		sendUsageError(c, "remove_problem")
	}
}

func routerClearClass(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		// This command requires confirmation
		// confirmationHandler.go will handle it
		c.Dispatcher.Send(dispatcher.MakeConfirmationMessage(
			c.Message.ChannelID,
			c.Message.ID,
			GetConfirmationMessage("!clear_class", c.Args),
		))
	default:
		sendUsageError(c, "clear_class")
	}
}

func routerCreateClass(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1, 2:
		sendUsageError(c, "create_class")
	default:
		admin.HandleCreateClass(c)
	}
}

func routerGetUsersInClass(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		general.HandleGetUsersInClass(c)
	default:
		sendUsageError(c, "users")
	}
}

func routerGetAllClasses(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		general.HandleGetAllClasses(c)
	default:
		sendUsageError(c, "classes")
	}
}

func routerUpdateProblem(c *context.ContextMessage) {
	if len(c.Args) >= 7 {
		mentor.HandleUpdateProblem(c)
	} else {
		sendUsageError(c, "update_problem")
	}
}

func routerGetMyClasses(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		general.HandleGetMyClasses(c)
	default:
		sendUsageError(c, "my_classes")
	}
}

func routerGetClassesOfUser(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		general.HandleGetClassesOfUsername(c)
	default:
		sendUsageError(c, "classes_of")
	}
}

func routerGetMyUsername(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		general.HandleGetMyUsername(c)
	default:
		sendUsageError(c, "me")
	}
}

func routerChangeUsername(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		general.HandleChangeUsername(c)
	default:
		sendUsageError(c, "change_username")
	}
}

func routerExterminate(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		// This command requires confirmation
		// confirmationHandler.go will handle it
		c.Dispatcher.Send(dispatcher.MakeConfirmationMessage(
			c.Message.ChannelID,
			c.Message.ID,
			GetConfirmationMessage("!exterminate", c.Args),
		))
	default:
		sendUsageError(c, "exterminate")
	}
}

func routerExport(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		admin.HandleExportClass(c)
	default:
		sendUsageError(c, "export")
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
