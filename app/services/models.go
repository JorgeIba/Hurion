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

// CellMetadata contains semantic information about a table cell
type CellMetadata struct {
	IsAC          bool
	IsProblemName bool
	IsUsername    bool
}

// RowMetadata contains semantic information for a row
type RowMetadata struct {
	IsTopicHeader bool
	Problem       *data.Problem  // nil for topic headers, contains problem reference for problem rows
	CellMetadata  []CellMetadata
}

// TableScore represents a formatted score table for display or export
type TableScore struct {
	Class       string
	Header      []string
	Table       [][]string
	RowMetadata []RowMetadata // Semantic metadata for each row
}

type Class struct {
	ClassCode string
	Name      string
	Users     []data.User
	Score     TableScore
}
