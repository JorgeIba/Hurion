package services

import "github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"

// CreateProblemParams represents the parameters needed to create a new problem
type CreateProblemParams struct {
	Name       string
	URL        string
	ClassCode  string
	Difficulty *string
	Topic      *string
}

// ProblemsByTopic represents a group of problems belonging to the same topic
type ProblemsByTopic struct {
	Topic    string
	Problems []data.Problem
}

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

// Score represents the complete score information for a class
// Contains both user scores and problems grouped by topic
type Score struct {
	UserScores      []PairUserProblemScore
	ProblemsByTopic []ProblemsByTopic
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
