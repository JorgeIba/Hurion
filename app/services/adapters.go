package services

import (
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/excelizer"
)

// ============================================================================
// Adapters to Excelizer types
// ============================================================================

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

// ============================================================================
// Adapters to TableScore (for display/export)
// ============================================================================

// ScoreToTableScore converts a Score struct into a TableScore with semantic metadata
func ScoreToTableScore(score Score) TableScore {
	// Build headers: "Problem \\ Username" followed by usernames with counts
	headers := []string{"Problem \\ Username"}
	for _, entry := range score.UserScores {
		headers = append(headers, fmt.Sprintf("%s (%d)", entry.User.UserName, entry.CountProblems))
	}

	var table [][]string
	var rowMetadata []RowMetadata

	// Iterate through problems grouped by topic
	for _, topicGroup := range score.ProblemsByTopic {
		// Add topic header row
		topicRow := make([]string, len(headers))
		topicRow[0] = topicGroup.Topic
		for i := 1; i < len(topicRow); i++ {
			topicRow[i] = "" // Empty cells for topic header
		}
		table = append(table, topicRow)

		// Topic header metadata - no cell metadata needed, whole row is special
		rowMetadata = append(rowMetadata, RowMetadata{
			IsTopicHeader: true,
			Problem:       nil,
			CellMetadata:  nil,
		})

		// Add problem rows for this topic
		for _, problem := range topicGroup.Problems {
			row := []string{problem.Name}
			cellMetadata := []CellMetadata{
				{IsProblemName: true, IsAC: false, IsUsername: false}, // First cell is problem name
			}

			for _, userScore := range score.UserScores {
				if v, exists := userScore.Problems[problem]; exists && v {
					row = append(row, "AC")
					cellMetadata = append(cellMetadata, CellMetadata{
						IsAC:          true,
						IsProblemName: false,
						IsUsername:    false,
					})
				} else {
					row = append(row, " ")
					cellMetadata = append(cellMetadata, CellMetadata{
						IsAC:          false,
						IsProblemName: false,
						IsUsername:    false,
					})
				}
			}
			table = append(table, row)

			// Problem row metadata with reference to the problem
			rowMetadata = append(rowMetadata, RowMetadata{
				IsTopicHeader: false,
				Problem:       &problem, // Reference to problem (can access difficulty, etc.)
				CellMetadata:  cellMetadata,
			})
		}
	}

	return TableScore{
		Header:      headers,
		Table:       table,
		RowMetadata: rowMetadata,
	}
}
