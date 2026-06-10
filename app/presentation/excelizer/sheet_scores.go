package excelizer

import (
	"github.com/JorgeIba/Hurion/app/customerr/presentationerr"
	"github.com/JorgeIba/Hurion/app/presentation"
	"github.com/xuri/excelize/v2"
)

func writeScoresData(f *excelize.File, sheetName string, tableScore presentation.TableScore) error {
	// Write headers (first row)
	for colIdx, header := range tableScore.Header {
		cell := makeCellRef(colIdx, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// Write table data (rows starting from row 2)
	for rowIdx, row := range tableScore.Table {
		excelRow := rowIdx + 2 // Excel rows are 1-indexed, and we start from row 2
		for colIdx, cellValue := range row {
			cell := makeCellRef(colIdx, excelRow)
			f.SetCellValue(sheetName, cell, cellValue)
		}
	}

	return nil
}

// AddScoresSheetFromTableScore creates a scores sheet using a TableScore with metadata
func AddScoresSheetFromTableScore(f *excelize.File, sheetName string, tableScore presentation.TableScore) error {
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return presentationerr.NewDataWritingError("failed to create sheet "+sheetName, err)
	}

	f.SetActiveSheet(index)

	err = writeScoresData(f, sheetName, tableScore)
	if err != nil {
		return presentationerr.NewDataWritingError("failed to write scores data", err)
	}

	return formatScoresSheet(f, sheetName, tableScore)
}
