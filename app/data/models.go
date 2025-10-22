package data

import (
	"fmt"
	"time"
)

type Problem struct {
	ID         int64
	Name       string
	ClassID    int64
	URL        string
	Difficulty *string // Optional field
	Topic      *string // Optional field
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
	UserID   string
	UserName string
}

type PendingConfirmation struct {
	MessageID string
	UserID    string
	ProblemID int64
}

type Difficulty string

const (
	DifficultyDefault Difficulty = "Not specified"
	DifficultyEasy    Difficulty = "Easy"
	DifficultyMedium  Difficulty = "Medium"
	DifficultyHard    Difficulty = "Hard"
)

// Default values for optional Problem fields
const (
	DefaultTopic = "General"
)

func (d Difficulty) String() string {
	return string(d)
}

func IsValidDifficulty(d string) bool {
	switch Difficulty(d) {
	case DifficultyDefault, DifficultyEasy, DifficultyMedium, DifficultyHard:
		return true
	default:
		return false
	}
}

func GetValidDifficulties() []Difficulty {
	return []Difficulty{DifficultyDefault, DifficultyEasy, DifficultyMedium, DifficultyHard}
}

func ParseDifficulty(s string) (Difficulty, error) {
	if IsValidDifficulty(s) {
		return Difficulty(s), nil
	}
	return "", fmt.Errorf("invalid difficulty: %s. Valid values: Not specified, Easy, Medium, Hard", s)
}

func (p *Problem) GetDifficulty() Difficulty {
	if p.Difficulty == nil {
		return DifficultyDefault
	}
	return Difficulty(*p.Difficulty)
}

func (p *Problem) GetTopic() string {
	if p.Topic == nil {
		return DefaultTopic
	}
	return *p.Topic
}
