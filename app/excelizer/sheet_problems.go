package excelizer

import (
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/xuri/excelize/v2"
)

func writeProblemsData(f *excelize.File, sheetName string, problems []data.Problem) error {
	// Write headers
	headers := []string{"Problem ID", "Problem Name", "URL"}
	for i, header := range headers {
		cell := fmt.Sprintf("%c%d", FirstColumn+i, HeaderRow)
		f.SetCellValue(sheetName, cell, header)
	}

	// Write problem data
	for i, problem := range problems {
		row := i + FirstDataRow

		// Problem ID column (A)
		cellProblemID := fmt.Sprintf("A%d", row)
		f.SetCellValue(sheetName, cellProblemID, problem.ID)

		// Problem Name column (B)
		cellProblemName := fmt.Sprintf("B%d", row)
		f.SetCellValue(sheetName, cellProblemName, problem.Name)

		// URL column (C) with hyperlink
		cellURL := fmt.Sprintf("C%d", row)
		f.SetCellValue(sheetName, cellURL, problem.URL)
		f.SetCellHyperLink(sheetName, cellURL, problem.URL, "External")
	}

	return nil
}

// AddProblemsSheet adds a new sheet with a list of problems
func AddProblemsSheet(f *excelize.File, sheetName string, problems []data.Problem) error {
	// Create new sheet
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("failed to create sheet %s: %w", sheetName, err)
	}

	// Set the sheet as active
	f.SetActiveSheet(index)

	// Write data
	err = writeProblemsData(f, sheetName, problems)
	if err != nil {
		return fmt.Errorf("failed to write problems data: %w", err)
	}

	// Apply formatting
	return formatProblemsList(f, sheetName, len(problems))
}
