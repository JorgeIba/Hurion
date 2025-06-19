package bot

import (
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/admin"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/enrolled"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/general"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/mentor"
)

type CommandRouter func(c *context.ContextMessage)

var commandMap = map[string]CommandRouter{
	"!bind":           routerBind,
	"!register":       routerRegister,
	"!deregister":     routerDeregister,
	"!list":           routerList,
	"!score":          routerScore,
	"!ac":             routerProposeAC,
	"!undo_ac":        routerUndoAC,
	"!add_problem":    routerAddProblem,
	"!remove_problem": routerRemoveProblem,
	"!clear_class":    routerClearClass,
	"!create_class":   routerCreateClass,
	"!users":          routerGetUsersInClass,
	"!classes":        routerGetAllClasses,
	"!update_problem": routerUpdateProblem,
	"!my_classes":     routerGetMyClasses,
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
	case 1, 2:
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
		admin.HandleClearClass(c)
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
	case 1, 2, 3, 4:
		fmt.Println(NotEnough)
	case 5:
		mentor.HandleUpdateProblem(c)
	default:
		fmt.Println(LimitExceed)
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
