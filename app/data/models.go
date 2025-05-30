package data

import (
	"time"
)

type Problem struct {
	ID      int64
	Name    string
	ClassID int64
	URL     string
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

type User struct {
	DiscordID string
	UserName  string
}

type PendingConfirmation struct {
	MessageID string
	UserID    string
	ProblemID int64
}
