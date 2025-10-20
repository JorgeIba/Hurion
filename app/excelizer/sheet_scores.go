package excelizer

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

func writeScoresData(f *excelize.File, sheetName string, scores []UserScore, problems []Problem) error {
	// Row 1: User names (starting from column B)
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", ScoresHeaderRow), "") // Empty cell A1
	for i, userScore := range scores {
		col := SecondColumn + i // Start from column B
		cell := fmt.Sprintf("%c%d", col, ScoresHeaderRow)
		f.SetCellValue(sheetName, cell, userScore.User.Name)
	}

	// Row 2: Total problems solved by each user (starting from column B)
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", ScoresTotalsRow), TotalSolvedLabel)
	for i, userScore := range scores {
		col := SecondColumn + i // Start from column B
		cell := fmt.Sprintf("%c%d", col, ScoresTotalsRow)
		f.SetCellValue(sheetName, cell, userScore.CountProblems)
	}

	for i, problem := range problems {
		row := i + ScoresFirstDataRow

		// Problem name in column A
		cellProblemName := fmt.Sprintf("A%d", row)
		f.SetCellValue(sheetName, cellProblemName, problem.Name)

		// User columns (B onwards) - show AC if solved, leave empty if not
		for j, userScore := range scores {
			col := SecondColumn + j // Start from column B
			cell := fmt.Sprintf("%c%d", col, row)

			if solved, exists := userScore.Problems[problem]; exists && solved {
				f.SetCellValue(sheetName, cell, StatusAccepted)
			} else {
				f.SetCellValue(sheetName, cell, "") // Leave empty
			}
		}
	}

	return nil
}

func AddScoresSheet(f *excelize.File, sheetName string, scores []UserScore, problems []Problem) error {
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("failed to create sheet %s: %w", sheetName, err)
	}

	f.SetActiveSheet(index)

	err = writeScoresData(f, sheetName, scores, problems)
	if err != nil {
		return fmt.Errorf("failed to write scores data: %w", err)
	}

	return formatScore(f, sheetName, len(problems), len(scores))
}
