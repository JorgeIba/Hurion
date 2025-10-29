package excelizer

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

func writeProblemsData(f *excelize.File, sheetName string, problems []Problem) ([]int, error) {
	// Write headers in row 1
	headers := []string{"ID", "Nombre", "URL", "Dificultad"}
	for i, header := range headers {
		cell := fmt.Sprintf("%c%d", FirstColumn+i, HeaderRow)
		f.SetCellValue(sheetName, cell, header)
	}

	currentRow := FirstDataRow
	var currentTopic string
	topicRows := []int{} // Track which rows are topic headers

	for _, problem := range problems {
		// If topic changes, add a topic header row
		if problem.Topic != currentTopic {
			currentTopic = problem.Topic

			// Write topic in first column and merge across all 4 columns (A to D)
			topicCell := fmt.Sprintf("A%d", currentRow)
			f.SetCellValue(sheetName, topicCell, currentTopic)

			// Merge cells from A to D for this row
			f.MergeCell(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("D%d", currentRow))

			topicRows = append(topicRows, currentRow)
			currentRow++
		}

		// Write problem data
		// Column A: Problem ID
		cellProblemID := fmt.Sprintf("A%d", currentRow)
		f.SetCellValue(sheetName, cellProblemID, problem.ID)

		// Column B: Problem Name
		cellProblemName := fmt.Sprintf("B%d", currentRow)
		f.SetCellValue(sheetName, cellProblemName, problem.Name)

		// Column C: URL with hyperlink
		cellURL := fmt.Sprintf("C%d", currentRow)
		f.SetCellValue(sheetName, cellURL, problem.URL)
		f.SetCellHyperLink(sheetName, cellURL, problem.URL, "External")

		// Column D: Difficulty (empty, will be colored)
		cellDifficulty := fmt.Sprintf("D%d", currentRow)
		f.SetCellValue(sheetName, cellDifficulty, "")

		currentRow++
	}

	return topicRows, nil
}

// AddProblemsSheet adds a new sheet with a list of problems
func AddProblemsSheet(f *excelize.File, sheetName string, problems []Problem) error {
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("failed to create sheet %s: %w", sheetName, err)
	}

	f.SetActiveSheet(index)

	topicRows, err := writeProblemsData(f, sheetName, problems)
	if err != nil {
		return fmt.Errorf("failed to write problems data: %w", err)
	}

	return formatProblemsList(f, sheetName, problems, topicRows)
}
