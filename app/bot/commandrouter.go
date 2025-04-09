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
	"!ac":             routerAC,
	"!undo_ac":        routerUndoAC,
	"!add_problem":    routerAddProblem,
	"!remove_problem": routerRemoveProblem,
	"!clear_class":    routerClearClass,
	"!create_class":   routerCreateClass,
}

const NotEnough = "Not enough parameters bro"
const LimitExceed = "Parameters Limit Exceed"

func routerBind(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		fmt.Println(NotEnough)
	default:
		general.BindUser(c)
	}
}

func routerRegister(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		general.RegisterUserInClass(c)
	case 3:
		mentor.HandleRegisterUser(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func routerDeregister(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		enrolled.DeregisterUser(c)
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
		general.GetListOfProblemsByClass(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func routerScore(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		general.GetScoreByClass(c)
	case 3:
		general.GetScoreByUserAndClass(c)
	}
}

func routerAC(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		fmt.Println(NotEnough)
	case 3:
		enrolled.AddAC(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func routerUndoAC(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		fmt.Println(NotEnough)
	case 3:
		enrolled.UndoAC(c)
	case 4:
		mentor.HandleUndoAC(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func routerAddProblem(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		fmt.Println(NotEnough)
	default:
		mentor.HandleAddProblem(c)
	}
}

func routerRemoveProblem(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		fmt.Println(NotEnough)
	case 3:
		mentor.HandleRemoveProblem(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func routerClearClass(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		admin.ClearClass(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func routerCreateClass(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		fmt.Println(NotEnough)
	default:
		admin.CreateClass(c)
	}
}
