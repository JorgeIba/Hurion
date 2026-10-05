package data

import (
	"time"
)

type Problem struct {
	ID         int64
	Name       string
	ClassID    int64
	URL        string
	Difficulty *string // Optional field
	Topic      *string // Optional field
	Priority   *string // Optional field; canonical code from the problem contract
	CreatedAt  time.Time
}

type SolvedProblem struct {
	UserID    string
	ProblemID int64
	SolvedAt  time.Time
}

type Class struct {
	ID        int64
	ClassCode string
	Name      string
}

// Classroom represents a location (e.g., Discord Channel) where a class is taught.
type Classroom struct {
	RoomID    string
	ClassCode string
}

// ClassAccessRole represents a role (e.g., Discord Role) that grants access to a class.
type ClassAccessRole struct {
	ClassCode string
	RoleID    string
}

type User struct {
	UserID      string
	DisplayName string
}

type PendingConfirmation struct {
	MessageID string
	UserID    string
	ProblemID int64
}

// Default values for optional Problem fields
const (
	DefaultTopic = "General"
)

func (p *Problem) GetTopic() string {
	if p.Topic == nil {
		return DefaultTopic
	}
	return *p.Topic
}
