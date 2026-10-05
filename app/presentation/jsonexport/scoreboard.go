package jsonexport

import (
	"encoding/json"
	"slices"
	"time"

	"github.com/JorgeIba/Hurion/app/services/service_model"
)

type ScoreboardJSON struct {
	ClassCode string          `json:"classCode"`
	ClassName string          `json:"className"`
	Problems  []JSONProblem   `json:"problems"`
	Users     []JSONUserScore `json:"users"`
}

type JSONProblem struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	URL        string    `json:"url"`
	Difficulty *string   `json:"difficulty"`
	Priority   *string   `json:"priority"`
	Topic      string    `json:"topic"`
	CreatedAt  time.Time `json:"createdAt"`
}

type JSONUserScore struct {
	UserID      string  `json:"userId"`
	DisplayName string  `json:"displayName"`
	SolvedCount int     `json:"solvedCount"`
	Solved      []int64 `json:"solved"`
}

// FormatScoreboard converts domain score entities into JSON formatted bytes
func FormatScoreboard(classCode, className string, score service_model.Score) ([]byte, error) {
	problems := make([]JSONProblem, 0)
	for _, group := range score.ProblemsByTopic {
		for _, p := range group.Problems {
			problems = append(problems, JSONProblem{
				ID:         p.ID,
				Name:       p.Name,
				URL:        p.URL,
				Difficulty: p.Difficulty,
				Priority:   p.Priority,
				Topic:      p.GetTopic(),
				CreatedAt:  p.CreatedAt.UTC(),
			})
		}
	}

	users := make([]JSONUserScore, 0)
	for _, uScore := range score.UserScores {
		solved := make([]int64, 0)
		for pID, isSolved := range uScore.Problems {
			if isSolved {
				solved = append(solved, pID)
			}
		}
		slices.Sort(solved)
		users = append(users, JSONUserScore{
			UserID:      uScore.User.UserID,
			DisplayName: uScore.User.DisplayName,
			SolvedCount: uScore.CountProblems,
			Solved:      solved,
		})
	}

	jsonObj := ScoreboardJSON{
		ClassCode: classCode,
		ClassName: className,
		Problems:  problems,
		Users:     users,
	}

	return json.MarshalIndent(jsonObj, "", "  ")
}
