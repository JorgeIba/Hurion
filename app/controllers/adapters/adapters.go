package adapters

import (
	"fmt"

	"github.com/JorgeIba/Hurion/app/presentation"
	"github.com/JorgeIba/Hurion/app/presentation/excelizer"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// ============================================================================
// Internal Mappers (Private)
// ============================================================================

func mapServiceProblemToPresentationProblem(p service_model.Problem) presentation.Problem {
	return presentation.Problem{
		ID:         p.ID,
		Name:       p.Name,
		URL:        p.URL,
		Difficulty: p.Difficulty,
		Priority:   p.Priority,
		Topic:      p.GetTopic(),
	}
}

func mapServiceProblemsToPresentationProblems(problems []service_model.Problem) []presentation.Problem {
	result := make([]presentation.Problem, len(problems))
	for i, p := range problems {
		result[i] = mapServiceProblemToPresentationProblem(p)
	}
	return result
}

// ============================================================================
// Adapters to Excelizer Types
// ============================================================================

func sanitizeDisplayName(name string) string {
	if name == "" {
		return name
	}
	// Prefix with single quote if it starts with =, +, -, or @ to prevent formula injection
	firstChar := name[0]
	if firstChar == '=' || firstChar == '+' || firstChar == '-' || firstChar == '@' {
		return "'" + name
	}
	return name
}

// TransformToExcelizerUsers converts service layer users to excelizer users
func TransformToExcelizerUsers(users []service_model.User) []excelizer.User {
	result := make([]excelizer.User, len(users))
	for i, u := range users {
		result[i] = excelizer.User{
			ID:   u.UserID,
			Name: sanitizeDisplayName(u.DisplayName),
		}
	}
	return result
}

// TransformToExcelizerProblemsByTopic converts service layer ProblemsByTopic to excelizer ProblemsByTopic
func TransformToExcelizerProblemsByTopic(problemsByTopic []service_model.ProblemsByTopic) []excelizer.ProblemsByTopic {
	result := make([]excelizer.ProblemsByTopic, len(problemsByTopic))
	for i, group := range problemsByTopic {
		excelizerProblems := make([]excelizer.Problem, len(group.Problems))
		for j, p := range group.Problems {
			excelizerProblems[j] = excelizer.Problem{
				ID:         p.ID,
				Name:       p.Name,
				URL:        p.URL,
				Difficulty: p.Difficulty,
				Priority:   p.Priority,
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

// TransformToPresentationProblemsByTopic converts service layer ProblemsByTopic to presentation ProblemsByTopic
func TransformToPresentationProblemsByTopic(problemsByTopic []service_model.ProblemsByTopic) []presentation.ProblemsByTopic {
	result := make([]presentation.ProblemsByTopic, len(problemsByTopic))
	for i, group := range problemsByTopic {
		result[i] = presentation.ProblemsByTopic{
			Topic:    group.Topic,
			Problems: mapServiceProblemsToPresentationProblems(group.Problems),
		}
	}
	return result
}

// ============================================================================
// Adapters to Presentation Layer (TableScore)
// ============================================================================

// ScoreToTableScore converts a Score struct into a TableScore with semantic metadata
func ScoreToTableScore(score service_model.Score) presentation.TableScore {
	// Build headers: "(ID) Problem \\ Display Name" followed by user names
	headers := []string{"(ID) Problem \\ User"}
	for _, entry := range score.UserScores {
		name := sanitizeDisplayName(entry.User.DisplayName)
		if name == "" {
			name = entry.User.UserID
		}
		headers = append(headers, name)
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
			row := []string{fmt.Sprintf("(%d) %s [Priority: %s]", problem.ID, problem.Name, presentation.PriorityLabel(problem.Priority))}
			cellMetadata := []presentation.CellMetadata{
				{IsProblemName: true, IsAC: false, IsUsername: false, IsTotal: false}, // First cell is problem name
			}

			for _, userScore := range score.UserScores {
				if v, exists := userScore.Problems[problem.ID]; exists && v {
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

			// Map service problem to presentation problem
			presProblem := mapServiceProblemToPresentationProblem(problem)

			// Problem row metadata with reference to the problem
			rowMetadata = append(rowMetadata, presentation.RowMetadata{
				IsTopicHeader: false,
				IsTotalsRow:   false,
				Problem:       &presProblem,
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
