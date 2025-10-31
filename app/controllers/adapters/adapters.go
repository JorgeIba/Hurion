package adapters

import (
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/presentation"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/presentation/excelizer"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services"
)

// ============================================================================
// Adapters to Excelizer Types
// ============================================================================

// TransformToExcelizerUsers converts data layer users to excelizer users
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

// TransformToExcelizerProblemsByTopic converts service layer ProblemsByTopic to excelizer ProblemsByTopic
func TransformToExcelizerProblemsByTopic(problemsByTopic []services.ProblemsByTopic) []excelizer.ProblemsByTopic {
	result := make([]excelizer.ProblemsByTopic, len(problemsByTopic))
	for i, group := range problemsByTopic {
		excelizerProblems := make([]excelizer.Problem, len(group.Problems))
		for j, p := range group.Problems {
			excelizerProblems[j] = excelizer.Problem{
				ID:         p.ID,
				Name:       p.Name,
				URL:        p.URL,
				Difficulty: string(p.GetDifficulty()),
				Topic:      p.GetTopic(),
			}
		}
		result[i] = excelizer.ProblemsByTopic{
			Topic:    group.Topic,
			Problems: excelizerProblems,
		}
	}
	return result
}

// ============================================================================
// Adapters to Presentation Layer (TableScore)
// ============================================================================

// Converts a Score struct into a TableScore with semantic metadata
func ScoreToTableScore(score services.Score) presentation.TableScore {
	// Build headers: "Problem \\ Username" followed by usernames
	headers := []string{"Problem \\ Username"}
	for _, entry := range score.UserScores {
		headers = append(headers, entry.User.UserName)
	}

	var table [][]string
	var rowMetadata []presentation.RowMetadata

	// Add Total row (second row in the sheet)
	totalsRow := []string{"Total"}
	totalsCellMetadata := []presentation.CellMetadata{
		{IsProblemName: false, IsAC: false, IsUsername: false, IsTotal: true},
	}
	for _, entry := range score.UserScores {
		totalsRow = append(totalsRow, fmt.Sprintf("%d", entry.CountProblems))
		totalsCellMetadata = append(totalsCellMetadata, presentation.CellMetadata{
			IsAC:          false,
			IsProblemName: false,
			IsUsername:    false,
			IsTotal:       true,
		})
	}
	table = append(table, totalsRow)
	rowMetadata = append(rowMetadata, presentation.RowMetadata{
		IsTopicHeader: false,
		IsTotalsRow:   true,
		Problem:       nil,
		CellMetadata:  totalsCellMetadata,
	})

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
		rowMetadata = append(rowMetadata, presentation.RowMetadata{
			IsTopicHeader: true,
			IsTotalsRow:   false,
			Problem:       nil,
			CellMetadata:  nil,
		})

		// Add problem rows for this topic
		for _, problem := range topicGroup.Problems {
			row := []string{problem.Name}
			cellMetadata := []presentation.CellMetadata{
				{IsProblemName: true, IsAC: false, IsUsername: false, IsTotal: false}, // First cell is problem name
			}

			for _, userScore := range score.UserScores {
				if v, exists := userScore.Problems[problem]; exists && v {
					row = append(row, "AC")
					cellMetadata = append(cellMetadata, presentation.CellMetadata{
						IsAC:          true,
						IsProblemName: false,
						IsUsername:    false,
						IsTotal:       false,
					})
				} else {
					row = append(row, " ")
					cellMetadata = append(cellMetadata, presentation.CellMetadata{
						IsAC:          false,
						IsProblemName: false,
						IsUsername:    false,
						IsTotal:       false,
					})
				}
			}
			table = append(table, row)

			// Problem row metadata with reference to the problem
			rowMetadata = append(rowMetadata, presentation.RowMetadata{
				IsTopicHeader: false,
				IsTotalsRow:   false,
				Problem:       &problem, // Reference to problem (can access difficulty, etc.)
				CellMetadata:  cellMetadata,
			})
		}
	}

	return presentation.TableScore{
		Header:      headers,
		Table:       table,
		RowMetadata: rowMetadata,
	}
}
