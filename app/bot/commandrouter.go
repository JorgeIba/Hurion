package bot

import (
	"fmt"

	"github.com/JorgeIba/Hurion/app/context"
	"github.com/JorgeIba/Hurion/app/controllers/admin"
	"github.com/JorgeIba/Hurion/app/controllers/enrolled"
	"github.com/JorgeIba/Hurion/app/controllers/general"
	"github.com/JorgeIba/Hurion/app/controllers/mentor"
	"github.com/JorgeIba/Hurion/app/dispatcher"
)

type CommandRouter func(c *context.ContextMessage)

var commandMap = map[string]CommandRouter{
	Help.String():             routerHelp,
	List.String():             routerList,
	Scores.String():           routerScores,
	Score.String():            routerScore,
	ScoreImage.String():       routerScoreImage,
	ScoreOf.String():          routerScoreOf,
	AC.String():               routerProposeAC,
	UndoAC.String():           routerUndoAC,
	AddProblem.String():       routerAddProblem,
	RemoveProblem.String():    routerRemoveProblem,
	ClearClass.String():       routerClearClass,
	CreateClass.String():      routerCreateClass,
	LinkRoles.String():        routerLinkRoles,
	LinkRooms.String():        routerLinkRooms,
	GetRoles.String():         routerGetClassRoles,
	GetChannels.String():      routerGetClassRooms,
	Users.String():            routerGetUsersInClass,
	Classes.String():          routerGetAllClasses,
	UpdateProblem.String():    routerUpdateProblem,
	MyClasses.String():        routerGetMyClasses,
	ClassesOf.String():        routerGetClassesOfUser,
	Exterminate.String():      routerExterminate,
	ExterminateClass.String(): routerExterminateClass,
	Export.String():           routerExport,
}

func sendUsageError(c *context.ContextMessage, command string) {
	message := fmt.Sprintf("Mmmm Invalid usage bro. Check `!help %s` for more info.", command)
	c.Dispatcher.Send(
		dispatcher.MakeSimpleMessage(c.Message.ChannelID, message),
	)
}

func sendConfirmationMessage(c *context.ContextMessage, command Command) {
	confirmationArgs, ok := GetConfirmationArgs(c, command.String())
	if !ok {
		sendUsageError(c, command.Name())
		return
	}

	confirmationContext := *c
	confirmationContext.Args = append([]string{}, confirmationArgs...)

	c.Dispatcher.Send(dispatcher.MakeConfirmationMessageWithArgs(
		c.Message.ChannelID,
		c.Message.ID,
		GetConfirmationMessage(&confirmationContext, command.String()),
		confirmationArgs,
	))
}

func routerHelp(c *context.ContextMessage) {
	HandleHelp(c)
}

func routerList(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1, 2:
		general.HandleGetListOfProblems(c)
	default:
		sendUsageError(c, List.Name())
	}
}

func routerScore(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1, 2:
		general.HandleGetScoreLink(c)
	default:
		sendUsageError(c, Score.Name())
	}
}

func routerScoreImage(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1, 2:
		general.HandleGetScoreImage(c)
	default:
		sendUsageError(c, ScoreImage.Name())
	}
}

func routerScores(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		general.HandleGetScoreAllClasses(c)
	default:
		sendUsageError(c, Scores.Name())
	}
}

func routerScoreOf(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2, 3:
		general.HandleGetScoreByUserAndClass(c)
	default:
		sendUsageError(c, ScoreOf.Name())
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
	if len(c.Args) < 6 {
		sendUsageError(c, AddProblem.Name())
	} else {
		mentor.HandleAddProblem(c)
	}
}

func routerRemoveProblem(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		// This command requires confirmation
		// confirmationHandler.go will handle it
		sendConfirmationMessage(c, RemoveProblem)
	default:
		sendUsageError(c, RemoveProblem.Name())
	}
}

func routerClearClass(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1, 2:
		// This command requires confirmation
		// confirmationHandler.go will handle it
		sendConfirmationMessage(c, ClearClass)
	default:
		sendUsageError(c, ClearClass.Name())
	}
}

func routerExterminateClass(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		sendConfirmationMessage(c, ExterminateClass)
	default:
		sendUsageError(c, ExterminateClass.Name())
	}
}

func routerCreateClass(c *context.ContextMessage) {
	if len(c.Args) < 3 {
		sendUsageError(c, CreateClass.Name())
	} else {
		admin.HandleCreateClass(c)
	}
}

func routerGetUsersInClass(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1, 2:
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
		general.HandleGetClassesOfUser(c)
	default:
		sendUsageError(c, ClassesOf.Name())
	}
}

func routerExterminate(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		// This command requires confirmation
		// confirmationHandler.go will handle it
		sendConfirmationMessage(c, Exterminate)
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

func HandleHelp(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		embed := buildGeneralHelpMessage()
		c.Dispatcher.Send(
			dispatcher.MakeEmbedMessage(c.Message.ChannelID, embed),
		)
	case 2:
		command := c.Args[1]
		if data, exists := getCommandDataByName(command); exists {
			embed := buildCommandHelpMessage(data)
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

func routerLinkRoles(c *context.ContextMessage) {
	if len(c.Args) >= 3 {
		admin.HandleLinkRoles(c)
	} else {
		sendUsageError(c, LinkRoles.Name())
	}
}

func routerLinkRooms(c *context.ContextMessage) {
	if len(c.Args) >= 3 {
		admin.HandleLinkRooms(c)
	} else {
		sendUsageError(c, LinkRooms.Name())
	}
}

func routerGetClassRoles(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1, 2:
		general.HandleGetClassRoles(c)
	default:
		sendUsageError(c, GetRoles.Name())
	}
}

func routerGetClassRooms(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1, 2:
		general.HandleGetClassRooms(c)
	default:
		sendUsageError(c, GetChannels.Name())
	}
}
