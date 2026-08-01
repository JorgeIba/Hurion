package jsonexport

import (
	"encoding/json"

	"github.com/JorgeIba/Hurion/app/services/service_model"
)

type ScoreboardJSON struct {
	ClassCode string           `json:"classCode"`
	ClassName string           `json:"className"`
	Problems  []JSONProblem    `json:"problems"`
	Users     []JSONUserScore  `json:"users"`
}

type JSONProblem struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	URL        string `json:"url"`
	Difficulty string `json:"difficulty"`
	Topic      string `json:"topic"`
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
				Difficulty: p.GetDifficulty(),
				Topic:      p.GetTopic(),
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
