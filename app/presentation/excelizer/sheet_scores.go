package excelizer

import (
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/presentation"
	"github.com/xuri/excelize/v2"
)

func writeScoresData(f *excelize.File, sheetName string, tableScore presentation.TableScore) error {
	// Write headers (first row)
	for colIdx, header := range tableScore.Header {
		col := FirstColumn + colIdx
		cell := fmt.Sprintf("%c%d", col, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// Write table data (rows starting from row 2)
	for rowIdx, row := range tableScore.Table {
		excelRow := rowIdx + 2 // Excel rows are 1-indexed, and we start from row 2
		for colIdx, cellValue := range row {
			col := FirstColumn + colIdx
			cell := fmt.Sprintf("%c%d", col, excelRow)
			f.SetCellValue(sheetName, cell, cellValue)
		}
	}

	return nil
}

// AddScoresSheetFromTableScore creates a scores sheet using a TableScore with metadata
func AddScoresSheetFromTableScore(f *excelize.File, sheetName string, tableScore presentation.TableScore) error {
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("failed to create sheet %s: %w", sheetName, err)
	}

	f.SetActiveSheet(index)

	err = writeScoresData(f, sheetName, tableScore)
	if err != nil {
		return fmt.Errorf("failed to write scores data: %w", err)
	}

	return formatScoresSheet(f, sheetName, tableScore)
}
