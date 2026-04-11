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
