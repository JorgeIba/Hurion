package excelizer

import (
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services"
	"github.com/xuri/excelize/v2"
)

func writeScoresData(f *excelize.File, sheetName string, userScores []services.PairUserProblemScore) ([]data.Problem, error) {
	// Collect all problems from all users
	problemsMap := make(map[data.Problem]bool)
	for _, userScore := range userScores {
		for problem := range userScore.Problems {
			problemsMap[problem] = true
		}
	}

	// Convert to slice for consistent ordering
	var problems []data.Problem
	for problem := range problemsMap {
		problems = append(problems, problem)
	}

	// Row 1: User names (starting from column B)
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", ScoresHeaderRow), "") // Empty cell A1
	for i, userScore := range userScores {
		col := SecondColumn + i // Start from column B
		cell := fmt.Sprintf("%c%d", col, ScoresHeaderRow)
		f.SetCellValue(sheetName, cell, userScore.User.UserName)
	}

	// Row 2: Total problems solved by each user (starting from column B)
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", ScoresTotalsRow), TotalSolvedLabel)
	for i, userScore := range userScores {
		col := SecondColumn + i // Start from column B
		cell := fmt.Sprintf("%c%d", col, ScoresTotalsRow)
		f.SetCellValue(sheetName, cell, userScore.CountProblems)
	}

	// From row 3 onwards: Problem names in column A, AC/empty for each user
	for i, problem := range problems {
		row := i + ScoresFirstDataRow

		// Problem name in column A
		cellProblemName := fmt.Sprintf("A%d", row)
		f.SetCellValue(sheetName, cellProblemName, problem.Name)

		// User columns (B onwards) - show AC if solved, leave empty if not
		for j, userScore := range userScores {
			col := SecondColumn + j // Start from column B
			cell := fmt.Sprintf("%c%d", col, row)

			if solved, exists := userScore.Problems[problem]; exists && solved {
				f.SetCellValue(sheetName, cell, StatusAccepted)
			} else {
				f.SetCellValue(sheetName, cell, "") // Leave empty
			}
		}
	}

	return problems, nil
}

// AddScoresSheet adds a new sheet with user scores using PairUserProblemScore model with custom structure
func AddScoresSheet(f *excelize.File, sheetName string, userScores []services.PairUserProblemScore) error {
	// Create new sheet
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("failed to create sheet %s: %w", sheetName, err)
	}

	// Set the sheet as active
	f.SetActiveSheet(index)

	// Write data
	problems, err := writeScoresData(f, sheetName, userScores)
	if err != nil {
		return fmt.Errorf("failed to write scores data: %w", err)
	}

	// Apply formatting
	return formatScore(f, sheetName, len(problems), len(userScores))
}
