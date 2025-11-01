package helpers

import (
	"sort"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
)

// GetDifficultyOrder returns the sort order for a difficulty
// Easy=1, Medium=2, Hard=3, Not specified=4
func GetDifficultyOrder(difficulty data.Difficulty) int {
	switch difficulty {
	case data.DifficultyEasy:
		return 1
	case data.DifficultyMedium:
		return 2
	case data.DifficultyHard:
		return 3
	case data.DifficultyDefault:
		return 4
	default:
		return 5 // Unknown difficulties go last
	}
}

// SortProblemsByTopicAndDifficulty sorts problems by topic (alphabetically) and then by difficulty
func SortProblemsByTopicAndDifficulty(problems []data.Problem) {
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

// PairUserSolvedProblems pairs a user with their solved problems
type PairUserSolvedProblems struct {
	User           data.User
	SolvedProblems []data.SolvedProblem
}

// SortScore sorts users by number of solved problems (descending)
func SortScore(solvedProblemsByUsers map[data.User][]data.SolvedProblem) []PairUserSolvedProblems {
	var userScores []PairUserSolvedProblems

	for user, problems := range solvedProblemsByUsers {
		userScores = append(userScores, PairUserSolvedProblems{
			User:           user,
			SolvedProblems: problems,
		})
	}

	sort.Slice(userScores, func(i, j int) bool {
		return len(userScores[i].SolvedProblems) > len(userScores[j].SolvedProblems)
	})

	return userScores
}
