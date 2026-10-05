package excelizer

import (
	"github.com/JorgeIba/Hurion/app/customerr/presentationerr"
	"github.com/JorgeIba/Hurion/app/presentation"
	"github.com/xuri/excelize/v2"
)

// formatUsersList applies formatting to a users list sheet
func formatUsersList(f *excelize.File, sheetName string, userCount int) error {
	headerStyle, err := CreateHeaderStyle(f)
	if err != nil {
		return presentationerr.NewStyleCreationError("failed to create header style", err)
	}

	// Apply header style to first row
	f.SetCellStyle(sheetName, makeCellRef(0, HeaderRow), makeCellRef(1, HeaderRow), headerStyle)

	if userCount > 0 {
		coloredRowStyle, err := CreatePastelRowStyle(f)
		if err != nil {
			return presentationerr.NewStyleCreationError("failed to create colored row style", err)
		}

		plainRowStyle, err := CreatePlainCenteredStyle(f)
		if err != nil {
			return presentationerr.NewStyleCreationError("failed to create plain row style", err)
		}

		endRow := userCount + HeaderRow
		for row := FirstDataRow; row <= endRow; row++ {
			if row%2 == 0 {
				f.SetCellStyle(sheetName, makeCellRef(0, row), makeCellRef(1, row), coloredRowStyle)
			} else {
				f.SetCellStyle(sheetName, makeCellRef(0, row), makeCellRef(1, row), plainRowStyle)
			}
		}
	}

	err = autoFitColumns(f, sheetName, []string{"A", "B"})
	if err != nil {
		return presentationerr.NewFormatError("failed to auto-fit columns", err)
	}

	err = setFreezePanes(f, sheetName, NoFreeze, FreezeFirstRow, TopLeftCellOneRow, ActivePaneBottom)
	if err != nil {
		return presentationerr.NewFormatError("failed to set freeze panes", err)
	}

	return nil
}

func formatProblemsList(f *excelize.File, sheetName string, problemsByTopic []ProblemsByTopic) error {
	headerStyle, err := CreateHeaderStyle(f)
	if err != nil {
		return presentationerr.NewStyleCreationError("failed to create header style", err)
	}
	f.SetCellStyle(sheetName, makeCellRef(0, 1), makeCellRef(4, 1), headerStyle)

	// Styles for topic header, ID, Name, URL, Difficulty columns
	topicHeaderStyle, err := CreateTopicHeaderStyle(f)
	if err != nil {
		return presentationerr.NewStyleCreationError("failed to create topic header style", err)
	}

	idStyle, err := CreatePlainCenteredStyle(f)
	if err != nil {
		return presentationerr.NewStyleCreationError("failed to create ID style", err)
	}

	nameStyle, err := CreatePlainLeftAlignedStyle(f)
	if err != nil {
		return presentationerr.NewStyleCreationError("failed to create name style", err)
	}

	urlStyle, err := CreateHyperlinkStyle(f)
	if err != nil {
		return presentationerr.NewStyleCreationError("failed to create URL style", err)
	}

	currentRow := FirstDataRow

	// Iterate through topics
	for _, topicGroup := range problemsByTopic {
		// Apply topic header style
		f.SetCellStyle(sheetName, makeCellRef(0, currentRow), makeCellRef(4, currentRow), topicHeaderStyle)
		currentRow++

		// Apply styles to problems in this topic
		for _, problem := range topicGroup.Problems {
			// ID (centered)
			f.SetCellStyle(sheetName, makeCellRef(0, currentRow), makeCellRef(0, currentRow), idStyle)

			// Name (left aligned)
			f.SetCellStyle(sheetName, makeCellRef(1, currentRow), makeCellRef(1, currentRow), nameStyle)

			// URL (hyperlink style)
			f.SetCellStyle(sheetName, makeCellRef(2, currentRow), makeCellRef(2, currentRow), urlStyle)

			// Difficulty (colored background)
			difficultyStyle, err := CreateDifficultyStyle(f, problem.Difficulty)
			if err != nil {
				return presentationerr.NewStyleCreationError("failed to create difficulty style", err)
			}
			f.SetCellStyle(sheetName, makeCellRef(3, currentRow), makeCellRef(3, currentRow), difficultyStyle)
			f.SetCellStyle(sheetName, makeCellRef(4, currentRow), makeCellRef(4, currentRow), nameStyle)

			currentRow++
		}
	}

	// Auto-fit columns
	err = autoFitColumns(f, sheetName, []string{"A", "B", "C", "D", "E"})
	if err != nil {
		return presentationerr.NewFormatError("failed to auto-fit columns", err)
	}

	// Freeze first row (headers)
	err = setFreezePanes(f, sheetName, NoFreeze, FreezeFirstRow, TopLeftCellOneRow, ActivePaneBottom)
	if err != nil {
		return presentationerr.NewFormatError("failed to set freeze panes", err)
	}

	return nil
}

// formatScoresSheet applies styling to the scores sheet based on metadata
func formatScoresSheet(f *excelize.File, sheetName string, tableScore presentation.TableScore) error {
	// Create styles
	headerStyle, err := CreateHeaderStyle(f)
	if err != nil {
		return presentationerr.NewStyleCreationError("failed to create header style", err)
	}

	topicStyle, err := CreateTopicHeaderStyle(f)
	if err != nil {
		return presentationerr.NewStyleCreationError("failed to create topic header style", err)
	}

	acStyle, err := CreateACStyle(f)
	if err != nil {
		return presentationerr.NewStyleCreationError("failed to create AC style", err)
	}

	totalsStyle, err := CreateTotalsStyle(f)
	if err != nil {
		return presentationerr.NewStyleCreationError("failed to create totals style", err)
	}

	plainStyle, err := CreatePlainCenteredStyle(f)
	if err != nil {
		return presentationerr.NewStyleCreationError("failed to create plain style", err)
	}

	// Apply header style to first row (user IDs)
	for colIdx := range tableScore.Header {
		cell := makeCellRef(colIdx, 1)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	// Apply styles to data rows based on metadata
	for rowIdx, metadata := range tableScore.RowMetadata {
		excelRow := rowIdx + 2

		if metadata.IsTotalsRow {
			// Apply totals style to entire row (green medium)
			for colIdx := range tableScore.Table[rowIdx] {
				cell := makeCellRef(colIdx, excelRow)
				f.SetCellStyle(sheetName, cell, cell, totalsStyle)
			}
		} else if metadata.IsTopicHeader {
			// Apply topic header style to entire row
			for colIdx := range tableScore.Table[rowIdx] {
				cell := makeCellRef(colIdx, excelRow)
				f.SetCellStyle(sheetName, cell, cell, topicStyle)
			}
		} else {
			// Problem rows: apply styles based on cell metadata
			for colIdx, cellMeta := range metadata.CellMetadata {
				cell := makeCellRef(colIdx, excelRow)

				if cellMeta.IsAC {
					// AC cells - medium green
					f.SetCellStyle(sheetName, cell, cell, acStyle)
				} else {
					// Regular cells (including problem names)
					f.SetCellStyle(sheetName, cell, cell, plainStyle)
				}
			}
		}
	}

	// Auto-fit columns
	numCols := len(tableScore.Header)
	columns := make([]string, numCols)
	for i := 0; i < numCols; i++ {
		columns[i] = columnIndexToLetter(i)
	}
	err = autoFitColumns(f, sheetName, columns)
	if err != nil {
		return presentationerr.NewFormatError("failed to auto-fit columns", err)
	}

	// Freeze first row and first column (headers and problem names)
	err = setFreezePanes(f, sheetName, FreezeFirstColumn, FreezeFirstRow, TopLeftCellOneRowOneCol, ActivePaneBottomRight)
	if err != nil {
		return presentationerr.NewFormatError("failed to set freeze panes", err)
	}

	return nil
}
