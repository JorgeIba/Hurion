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
	"!bind":           handleBind,
	"!register":       handleRegister,
	"!deregister":     handleDeregister,
	"!list":           handleList,
	"!score":          handleScore,
	"!ac":             handleAC,
	"!undo_ac":        handleUndoAC,
	"!add_problem":    handleAddProblem,
	"!remove_problem": handleRemoveProblem,
	"!clear_class":    handleClearClass,
	"!create_class":   handleCreateClass,
}

const NotEnough = "Not enough parameters bro"
const LimitExceed = "Parameters Limit Exceed"

func handleBind(c *context.ContextMessage) {
	switch len(c.Args) {
	case 1:
		fmt.Println(NotEnough)
	default:
		general.BindUser(c)
	}
}

func handleRegister(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		general.RegisterUserInClass(c)
	case 3:
		mentor.RegisterUserInClassByUser(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleDeregister(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		enrolled.DeregisterUser(c)
	case 3:
		mentor.DeregisterUserByUserAndClass(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleList(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		general.GetListOfProblemsByClass(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleScore(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		general.GetScoreByClass(c)
	case 3:
		general.GetScoreByUserAndClass(c)
	}
}

func handleAC(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		fmt.Println(NotEnough)
	case 3:
		enrolled.AddAC(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleUndoAC(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		fmt.Println(NotEnough)
	case 3:
		enrolled.UndoAC(c)
	case 4:
		mentor.UndoACByUser(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleAddProblem(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		fmt.Println(NotEnough)
	case 3:
		mentor.AddProblem(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleRemoveProblem(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		fmt.Println(NotEnough)
	case 3:
		mentor.RemoveProblem(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleClearClass(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		admin.ClearClass(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleCreateClass(c *context.ContextMessage) {
	switch len(c.Args) {
	case 2:
		fmt.Println(NotEnough)
	default:
		admin.CreateClass(c)
	}
}
