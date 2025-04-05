package bot

import (
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/bot/contextmessage"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/admin"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/enrolled"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/general"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/mentor"
)

type CommandRouter func(c *contextmessage.ContextMessage)

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

func handleBind(c *contextmessage.ContextMessage) {
	switch len(c.Args) {
	case 1:
		fmt.Println(NotEnough)
	default:
		general.BindUser(c)
	}
}

func handleRegister(c *contextmessage.ContextMessage) {
	switch len(c.Args) {
	case 2:
		general.RegisterUserInClass(c)
	case 3:
		mentor.RegisterUserInClassByUser(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleDeregister(c *contextmessage.ContextMessage) {
	switch len(c.Args) {
	case 2:
		enrolled.DeregisterUser(c)
	case 3:
		mentor.DeregisterUserByUser(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleList(c *contextmessage.ContextMessage) {
	switch len(c.Args) {
	case 2:
		general.GetListOfProblemsByClass(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleScore(c *contextmessage.ContextMessage) {
	switch len(c.Args) {
	case 2:
		general.GetScoreByClass(c)
	case 3:
		general.GetScoreByUserAndClass(c)
	}
}

func handleAC(c *contextmessage.ContextMessage) {
	switch len(c.Args) {
	case 2:
		fmt.Println(NotEnough)
	case 3:
		enrolled.AddAC(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleUndoAC(c *contextmessage.ContextMessage) {
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

func handleAddProblem(c *contextmessage.ContextMessage) {
	switch len(c.Args) {
	case 2:
		fmt.Println(NotEnough)
	case 3:
		mentor.AddProblem(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleRemoveProblem(c *contextmessage.ContextMessage) {
	switch len(c.Args) {
	case 2:
		fmt.Println(NotEnough)
	case 3:
		mentor.RemoveProblem(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleClearClass(c *contextmessage.ContextMessage) {
	switch len(c.Args) {
	case 2:
		admin.ClearClass(c)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleCreateClass(c *contextmessage.ContextMessage) {
	switch len(c.Args) {
	case 2:
		fmt.Println(NotEnough)
	case 3:
		admin.CreateClass(c)
	default:
		fmt.Println(LimitExceed)
	}
}
