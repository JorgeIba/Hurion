package bot

import (
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/admin"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/enrolled"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/general"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/mentor"
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
		general.BindUser(s, m)
	}
}

func handleRegister(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	switch len(args) {
	case 2:
		general.RegisterUserInClass(s, m)
	case 3:
		mentor.RegisterUserInClassByUser(s, m)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleDeregister(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	switch len(args) {
	case 2:
		enrolled.DeregisterUser(s, m)
	case 3:
		mentor.DeregisterUserByUser(s, m)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleList(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	switch len(args) {
	case 2:
		general.GetListOfProblemsByClass(s, m)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleScore(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	switch len(args) {
	case 2:
		general.GetScoreByClass(s, m)
	case 3:
		general.GetScoreByUserAndClass(s, m)
	}
}

func handleAC(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	switch len(args) {
	case 2:
		fmt.Println(NotEnough)
	case 3:
		enrolled.AddAC(s, m)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleUndoAC(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	switch len(args) {
	case 2:
		fmt.Println(NotEnough)
	case 3:
		enrolled.UndoAC(s, m)
	case 4:
		mentor.UndoACByUser(s, m)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleAddProblem(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	switch len(args) {
	case 2:
		fmt.Println(NotEnough)
	case 3:
		mentor.AddProblem(s, m)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleRemoveProblem(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	switch len(args) {
	case 2:
		fmt.Println(NotEnough)
	case 3:
		mentor.RemoveProblem(s, m)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleClearClass(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	switch len(args) {
	case 2:
		admin.ClearClass(s, m)
	default:
		fmt.Println(LimitExceed)
	}
}

func handleCreateClass(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	switch len(args) {
	case 2:
		fmt.Println(NotEnough)
	case 3:
		admin.CreateClass(s, m)
	default:
		fmt.Println(LimitExceed)
	}
}
