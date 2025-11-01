package excelizer

import (
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/presentationerr"
	"github.com/xuri/excelize/v2"
)

func writeProblemsData(f *excelize.File, sheetName string, problemsByTopic []ProblemsByTopic) error {
	// Write headers in row 1
	headers := []string{"ID", "Nombre", "URL", "Dificultad"}
	for i, header := range headers {
		cell := makeCellRef(i, HeaderRow)
		f.SetCellValue(sheetName, cell, header)
	}

	currentRow := FirstDataRow

	// Iterate through topics
	for _, topicGroup := range problemsByTopic {
		// Add topic header row
		topicCell := makeCellRef(0, currentRow)
		f.SetCellValue(sheetName, topicCell, topicGroup.Topic)

		// Merge cells from A to D for this row
		f.MergeCell(sheetName, makeCellRef(0, currentRow), makeCellRef(3, currentRow))

		currentRow++

		// Write problems for this topic
		for _, problem := range topicGroup.Problems {
			// Column A: Problem ID
			cellProblemID := makeCellRef(0, currentRow)
			f.SetCellValue(sheetName, cellProblemID, problem.ID)

			// Column B: Problem Name
			cellProblemName := makeCellRef(1, currentRow)
			f.SetCellValue(sheetName, cellProblemName, problem.Name)

			// Column C: URL with hyperlink
			cellURL := makeCellRef(2, currentRow)
			f.SetCellValue(sheetName, cellURL, problem.URL)
			f.SetCellHyperLink(sheetName, cellURL, problem.URL, "External")

			// Column D: Difficulty (empty, will be colored)
			cellDifficulty := makeCellRef(3, currentRow)
			f.SetCellValue(sheetName, cellDifficulty, "")

			currentRow++
		}
	}

	return nil
}

// AddProblemsSheet adds a new sheet with problems grouped by topic
func AddProblemsSheet(f *excelize.File, sheetName string, problemsByTopic []ProblemsByTopic) error {
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return presentationerr.NewDataWritingError("failed to create sheet "+sheetName, err)
	}

	f.SetActiveSheet(index)

	err = writeProblemsData(f, sheetName, problemsByTopic)
	if err != nil {
		return presentationerr.NewDataWritingError("failed to write problems data", err)
	}

	return formatProblemsList(f, sheetName, problemsByTopic)
}
