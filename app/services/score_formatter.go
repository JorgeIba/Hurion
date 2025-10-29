package services

import "fmt"

// Converts a Score struct into a TableScore for display
func FormatScoreAsTable(score Score) TableScore {
	// Build headers: "Problem \\ Username" followed by usernames with counts
	headers := []string{"Problem \\ Username"}
	for _, entry := range score.UserScores {
		headers = append(headers, fmt.Sprintf("%s (%d)", entry.User.UserName, entry.CountProblems))
	}

	var table [][]string

	// Iterate through problems grouped by topic
	for _, topicGroup := range score.ProblemsByTopic {
		// Add topic header row
		topicRow := make([]string, len(headers))
		topicRow[0] = topicGroup.Topic
		for i := 1; i < len(topicRow); i++ {
			topicRow[i] = "" // Empty cells for topic header
		}
		table = append(table, topicRow)

		// Add problem rows for this topic
		for _, problem := range topicGroup.Problems {
			row := []string{problem.Name}
			for _, userScore := range score.UserScores {
				if v, exists := userScore.Problems[problem]; exists && v {
					row = append(row, "AC")
				} else {
					row = append(row, " ")
				}
			}
			table = append(table, row)
		}
	}

	return TableScore{Header: headers, Table: table}
}
