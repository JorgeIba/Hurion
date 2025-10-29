package services

import (
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/excelizer"
)

// Converts data layer users to excelizer users
func TransformToExcelizerUsers(users []data.User) []excelizer.User {
	result := make([]excelizer.User, len(users))
	for i, u := range users {
		result[i] = excelizer.User{
			ID:   u.UserID,
			Name: u.UserName,
		}
	}
	return result
}

// Converts data layer problems to excelizer problems
func TransformToExcelizerProblems(problems []data.Problem) []excelizer.Problem {
	result := make([]excelizer.Problem, len(problems))
	for i, p := range problems {
		result[i] = excelizer.Problem{
			ID:         p.ID,
			Name:       p.Name,
			URL:        p.URL,
			Difficulty: string(p.GetDifficulty()),
			Topic:      p.GetTopic(),
		}
	}
	return result
}

// Converts service layer scores to excelizer scores
// Requires problems list to properly map problem references
func TransformToExcelizerScores(scores []PairUserProblemScore, problems []data.Problem) []excelizer.UserScore {
	problemMap := make(map[data.Problem]excelizer.Problem)
	for _, p := range problems {
		problemMap[p] = excelizer.Problem{
			ID:         p.ID,
			Name:       p.Name,
			URL:        p.URL,
			Difficulty: string(p.GetDifficulty()),
			Topic:      p.GetTopic(),
		}
	}

	result := make([]excelizer.UserScore, len(scores))
	for i, score := range scores {
		// Transform user
		excelizerUser := excelizer.User{
			ID:   score.User.UserID,
			Name: score.User.UserName,
		}

		// Transform problems map from data.Problem -> excelizer.Problem
		excelizerProblems := make(map[excelizer.Problem]bool)
		for dataProblem, solved := range score.Problems {
			excelizerProblem := problemMap[dataProblem]
			excelizerProblems[excelizerProblem] = solved
		}

		result[i] = excelizer.UserScore{
			User:          excelizerUser,
			CountProblems: score.CountProblems,
			Problems:      excelizerProblems,
		}
	}
	return result
}
