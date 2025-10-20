package excelizer

import "sort"

func SortUsersAlphabetically(users []User) {
	sort.Slice(users, func(i, j int) bool {
		return users[i].Name < users[j].Name
	})
}

func SortProblemsByID(problems []Problem) {
	sort.Slice(problems, func(i, j int) bool {
		return problems[i].ID < problems[j].ID
	})
}

func SortProblemsByName(problems []Problem) {
	sort.Slice(problems, func(i, j int) bool {
		return problems[i].Name < problems[j].Name
	})
}

func SortScoresByTotalDescending(scores []UserScore) {
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].CountProblems > scores[j].CountProblems
	})
}

func SortScoresByTotalAscending(scores []UserScore) {
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].CountProblems < scores[j].CountProblems
	})
}

func SortScoresByUserName(scores []UserScore) {
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].User.Name < scores[j].User.Name
	})
}
