package helpers

import (
	"sort"

	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// SortProblemsByID establishes insertion order before topic grouping.
func SortProblemsByID(problems []service_model.Problem) {
	sort.Slice(problems, func(i, j int) bool {
		return problems[i].ID < problems[j].ID
	})
}

// SortScore sorts users by number of solved problems (descending)
// If problem counts are equal, it sorts by UserID (ascending) to ensure deterministic ranking
func SortScore(userScores []service_model.PairUserProblemScore) {
	sort.SliceStable(userScores, func(i, j int) bool {
		if userScores[i].CountProblems != userScores[j].CountProblems {
			return userScores[i].CountProblems > userScores[j].CountProblems
		}
		return userScores[i].User.UserID < userScores[j].User.UserID
	})
}
