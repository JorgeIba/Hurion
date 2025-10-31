package services

import (
	"fmt"
)

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
