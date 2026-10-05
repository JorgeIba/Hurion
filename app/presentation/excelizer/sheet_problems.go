package excelizer

import (
	"github.com/JorgeIba/Hurion/app/customerr/presentationerr"
	"github.com/JorgeIba/Hurion/app/presentation"
	"github.com/xuri/excelize/v2"
)

func writeProblemsData(f *excelize.File, sheetName string, problemsByTopic []ProblemsByTopic) error {
	// Write headers in row 1
	headers := []string{"ID", "Name", "URL", "Difficulty", "Priority"}
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

		f.MergeCell(sheetName, makeCellRef(0, currentRow), makeCellRef(len(headers)-1, currentRow))

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

			// Priority stays independent of difficulty colors.
			f.SetCellValue(sheetName, makeCellRef(4, currentRow), presentation.PriorityLabel(problem.Priority))

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
