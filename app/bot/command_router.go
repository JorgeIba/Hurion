package bot

import (
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers"
	"github.com/bwmarrin/discordgo"
)

type CommandRouter func(s *discordgo.Session, m *discordgo.MessageCreate, args []string)

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

func handleBind(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	switch len(args) {
	case 1:
		fmt.Println(NotEnough)
	default:
		controllers.BindUser(s, m)
	}
}

func handleRegister(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	switch len(args) {
	case 2:
		controllers.RegisterUserInClass(s, m)
	case 3:
		controllers.RegisterUserInClassByUser(s, m)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleDeregister(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	switch len(args) {
	case 2:
		controllers.DeregisterUser(s, m)
	case 3:
		controllers.DeregisterUserByUser(s, m)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleList(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	switch len(args) {
	case 2:
		controllers.GetListOfProblemsByClass(s, m)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleScore(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	switch len(args) {
	case 2:
		controllers.GetScoreByClass(s, m)
	case 3:
		controllers.GetScoreByUserAndClass(s, m)
	}
}

func handleAC(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	switch len(args) {
	case 2:
		fmt.Println(NotEnough)
	case 3:
		controllers.AddAC(s, m)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleUndoAC(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	switch len(args) {
	case 2:
		fmt.Println(NotEnough)
	case 3:
		controllers.UndoAC(s, m)
	case 4:
		controllers.UndoACByUser(s, m)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleAddProblem(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	switch len(args) {
	case 2:
		fmt.Println(NotEnough)
	case 3:
		controllers.AddProblem(s, m)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleRemoveProblem(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	switch len(args) {
	case 2:
		fmt.Println(NotEnough)
	case 3:
		controllers.RemoveProblem(s, m)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleClearClass(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	switch len(args) {
	case 2:
		controllers.ClearClass(s, m)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleCreateClass(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	switch len(args) {
	case 2:
		fmt.Println(NotEnough)
	case 3:
		controllers.CreateClass(s, m)
	default:
		fmt.Println(LimitExceed)
	}
}
