package excelizer

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

func writeProblemsData(f *excelize.File, sheetName string, problems []Problem) error {
	headers := []string{"Problem ID", "Problem Name", "URL"}
	for i, header := range headers {
		cell := fmt.Sprintf("%c%d", FirstColumn+i, HeaderRow)
		f.SetCellValue(sheetName, cell, header)
	}

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
func AddProblemsSheet(f *excelize.File, sheetName string, problems []Problem) error {
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("failed to create sheet %s: %w", sheetName, err)
	}

	f.SetActiveSheet(index)

	err = writeProblemsData(f, sheetName, problems)
	if err != nil {
		return fmt.Errorf("failed to write problems data: %w", err)
	}

	return formatProblemsList(f, sheetName, len(problems))
}
