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
	Help.String():           routerHelp,
	Bind.String():           routerBind,
	Register.String():       routerRegister,
	Deregister.String():     routerDeregister,
	List.String():           routerList,
	Score.String():          routerScore,
	AC.String():             routerProposeAC,
	UndoAC.String():         routerUndoAC,
	AddProblem.String():     routerAddProblem,
	RemoveProblem.String():  routerRemoveProblem,
	ClearClass.String():     routerClearClass,
	CreateClass.String():    routerCreateClass,
	Users.String():          routerGetUsersInClass,
	Classes.String():        routerGetAllClasses,
	UpdateProblem.String():  routerUpdateProblem,
	MyClasses.String():      routerGetMyClasses,
	ClassesOf.String():      routerGetClassesOfUser,
	Me.String():             routerGetMyUsername,
	ChangeUsername.String(): routerChangeUsername,
	Exterminate.String():    routerExterminate,
	Export.String():         routerExport,
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
		sendUsageError(c, Bind.Name())
	}
}

func routerRegister(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		general.HandleRegisterUserInClass(c)
	case 3:
		mentor.HandleRegisterUser(c)
	default:
		sendUsageError(c, Register.Name())
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
			GetConfirmationMessage(Deregister.String(), c.Args),
		))
	default:
		sendUsageError(c, Deregister.Name())
	}
}

func routerList(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		general.HandleGetListOfProblemsByClass(c)
	default:
		sendUsageError(c, List.Name())
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
		sendUsageError(c, AC.Name())
	}
}

func routerUndoAC(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		enrolled.HandleUndoAC(c)
	case 3:
		mentor.HandleUndoAC(c)
	default:
		sendUsageError(c, UndoAC.Name())
	}
}

func routerAddProblem(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1, 2, 3, 4, 5:
		fmt.Println(NotEnough)
		sendUsageError(c, AddProblem.Name())
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
			GetConfirmationMessage(RemoveProblem.String(), c.Args),
		))
	default:
		sendUsageError(c, RemoveProblem.Name())
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
			GetConfirmationMessage(ClearClass.String(), c.Args),
		))
	default:
		sendUsageError(c, ClearClass.Name())
	}
}

func routerCreateClass(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1, 2:
		sendUsageError(c, CreateClass.Name())
	default:
		admin.HandleCreateClass(c)
	}
}

func routerGetUsersInClass(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		general.HandleGetUsersInClass(c)
	default:
		sendUsageError(c, Users.Name())
	}
}

func routerGetAllClasses(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		general.HandleGetAllClasses(c)
	default:
		sendUsageError(c, Classes.Name())
	}
}

func routerUpdateProblem(c *context.ContextMessage) {
	if len(c.Args) >= 7 {
		mentor.HandleUpdateProblem(c)
	} else {
		sendUsageError(c, UpdateProblem.Name())
	}
}

func routerGetMyClasses(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		general.HandleGetMyClasses(c)
	default:
		sendUsageError(c, MyClasses.Name())
	}
}

func routerGetClassesOfUser(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		general.HandleGetClassesOfUsername(c)
	default:
		sendUsageError(c, ClassesOf.Name())
	}
}

func routerGetMyUsername(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		general.HandleGetMyUsername(c)
	default:
		sendUsageError(c, Me.Name())
	}
}

func routerChangeUsername(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		general.HandleChangeUsername(c)
	default:
		sendUsageError(c, ChangeUsername.Name())
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
			GetConfirmationMessage(Exterminate.String(), c.Args),
		))
	default:
		sendUsageError(c, Exterminate.Name())
	}
}

func routerExport(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		admin.HandleExportClass(c)
	default:
		sendUsageError(c, Export.Name())
	}
}

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
		if data, exists := getCommandDataByName(command); exists {
			embed := buildCommandHelpMessage(data)
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
