package services

import "github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"

// PairUserSolvedProblems represents a user and their solved problems
type PairUserSolvedProblems struct {
	User           data.User
	SolvedProblems []data.SolvedProblem
}

// PairUserProblemScore represents a user's score with problem completion status
type PairUserProblemScore struct {
	User          data.User
	CountProblems int
	Problems      map[data.Problem]bool
}

// TableScore represents a formatted score table for display or export
type TableScore struct {
	Class  string
	Header []string
	Table  [][]string
}

type Class struct {
	ClassCode string
	Name      string
	Users     []data.User
	Score     TableScore
}
