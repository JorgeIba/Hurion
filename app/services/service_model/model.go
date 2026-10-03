package service_model

import (
	"time"
)

// ============================================================================
// Core Service Models (Domain Entities)
// ============================================================================

// User represents a user in the system
type User struct {
	UserID      string
	DisplayName string
}

// Principal represents the user ACTING on the system right now,
// containing all their verified security credentials.
type Principal struct {
	User
	Roles []string
}

// Problem represents a problem within a class
type Problem struct {
	ID         int64
	Name       string
	ClassID    int64
	URL        string
	Difficulty *string
	Topic      *string
}

// Class represents a class or group of problems
type Class struct {
	ID        int64
	ClassCode string
	Name      string
}

// Classroom represents a location where a class is taught
type Classroom struct {
	RoomID    string
	ClassCode string
}

// ClassAccessRole represents a role that grants access to a class
type ClassAccessRole struct {
	RoleID    string
	ClassCode string
}

// SolvedProblem represents a record of a user solving a problem
type SolvedProblem struct {
	UserID    string
	ProblemID int64
	SolvedAt  time.Time
}

// PendingConfirmation represents a pending AC proposal awaiting mentor approval
type PendingConfirmation struct {
	MessageID string
	UserID    string
	ProblemID int64
}

// ============================================================================
// Request Parameters
// ============================================================================

// CreateProblemParams represents the parameters needed to create a new problem
type CreateProblemParams struct {
	Name       string
	URL        string
	ClassCode  string
	Difficulty *string
	Topic      *string
}

// UpdateProblemParams identifies the target and supplies all replacement values,
// including class membership. Nil Difficulty or Topic clears the stored value;
// omitted fields do not mean "leave unchanged". This type does not validate input.
type UpdateProblemParams struct {
	ProblemID  int64
	Name       string
	URL        string
	ClassID    int64
	Difficulty *string
	Topic      *string
}

// ============================================================================
// Composite Models (Used for presentation/business logic)
// ============================================================================

// ProblemsByTopic represents a group of problems belonging to the same topic
type ProblemsByTopic struct {
	Topic    string
	Problems []Problem
}

// PairUserSolvedProblems represents a user and their solved problems
type PairUserSolvedProblems struct {
	User           User
	SolvedProblems []SolvedProblem
}

// PairUserProblemScore represents a user's score with problem completion status
type PairUserProblemScore struct {
	User          User
	CountProblems int
	Problems      map[int64]bool // Map problem ID to solved status
}

// Score represents the complete score information for a class
// Contains both user scores and problems grouped by topic
type Score struct {
	UserScores      []PairUserProblemScore
	ProblemsByTopic []ProblemsByTopic
}

// ClassScore pairs a class's stable identity and display name with its score.
// A slice of ClassScore values preserves classes that share a display name and
// supports a deterministic presentation order.
type ClassScore struct {
	ClassCode string
	ClassName string
	Score     Score
}

const (
	DifficultyEasy    = "Easy"
	DifficultyMedium  = "Medium"
	DifficultyHard    = "Hard"
	DifficultyDefault = "Not specified"
	DefaultTopic      = "General"
)

func (p *Problem) GetDifficulty() string {
	if p.Difficulty == nil {
		return DifficultyDefault
	}
	return *p.Difficulty
}

func (p *Problem) GetTopic() string {
	if p.Topic == nil {
		return DefaultTopic
	}
	return *p.Topic
}
