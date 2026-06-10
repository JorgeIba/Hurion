package helpers

import (
	"sort"

	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// GetDifficultyOrder returns the sort order for a difficulty
// Easy=1, Medium=2, Hard=3, Not specified=4
func GetDifficultyOrder(difficulty string) int {
	switch difficulty {
	case service_model.DifficultyEasy:
		return 1
	case service_model.DifficultyMedium:
		return 2
	case service_model.DifficultyHard:
		return 3
	default:
		return 4 // Default
	}
}

// SortProblemsByTopicAndDifficulty sorts problems by topic (alphabetically) and then by difficulty
func SortProblemsByTopicAndDifficulty(problems []service_model.Problem) {
	sort.Slice(problems, func(i, j int) bool {
		topicI := problems[i].GetTopic()
		topicJ := problems[j].GetTopic()

		// First, sort by topic (alphabetically)
		if topicI != topicJ {
			return topicI < topicJ
		}

		// If topics are the same, sort by difficulty
		diffI := problems[i].GetDifficulty()
		diffJ := problems[j].GetDifficulty()
		return GetDifficultyOrder(diffI) < GetDifficultyOrder(diffJ)
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
