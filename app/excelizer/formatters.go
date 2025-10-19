package excelizer

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// formatUsersList applies formatting to a users list sheet
func formatUsersList(f *excelize.File, sheetName string, userCount int) error {
	headerStyle, err := CreateHeaderStyle(f)
	if err != nil {
		return fmt.Errorf("failed to create header style: %w", err)
	}

	// Apply header style to first row
	f.SetCellStyle(sheetName, fmt.Sprintf("A%d", HeaderRow), fmt.Sprintf("B%d", HeaderRow), headerStyle)

	if userCount > 0 {
		coloredRowStyle, err := CreatePastelRowStyle(f)
		if err != nil {
			return fmt.Errorf("failed to create colored row style: %w", err)
		}

		plainRowStyle, err := CreatePlainCenteredStyle(f)
		if err != nil {
			return fmt.Errorf("failed to create plain row style: %w", err)
		}

		endRow := userCount + HeaderRow
		for row := FirstDataRow; row <= endRow; row++ {
			if row%2 == 0 {
				f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row), coloredRowStyle)
			} else {
				f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row), plainRowStyle)
			}
		}
	}

	err = autoFitColumns(f, sheetName, []string{"A", "B"})
	if err != nil {
		return fmt.Errorf("failed to auto-fit columns: %w", err)
	}

	err = setFreezePanes(f, sheetName, NoFreeze, FreezeFirstRow, TopLeftCellOneRow, ActivePaneBottom)
	if err != nil {
		return fmt.Errorf("failed to set freeze panes: %w", err)
	}

	return nil
}

// formatProblemsList applies formatting to a problems list sheet
func formatProblemsList(f *excelize.File, sheetName string, problemCount int) error {
	headerStyle, err := CreateHeaderStyle(f)
	if err != nil {
		return fmt.Errorf("failed to create header style: %w", err)
	}

	f.SetCellStyle(sheetName, fmt.Sprintf("A%d", HeaderRow), fmt.Sprintf("C%d", HeaderRow), headerStyle)

	// First column with pastel green, name column left aligned, URL column with hyperlink style
	firstColumnStyle, err := CreatePastelColumnStyle(f)
	if err != nil {
		return fmt.Errorf("failed to create first column style: %w", err)
	}

	nameStyle, err := CreatePlainLeftAlignedStyle(f)
	if err != nil {
		return fmt.Errorf("failed to create name style: %w", err)
	}

	urlStyle, err := CreateHyperlinkStyle(f)
	if err != nil {
		return fmt.Errorf("failed to create URL style: %w", err)
	}

	endRow := problemCount + HeaderRow
	f.SetCellStyle(sheetName, fmt.Sprintf("A%d", FirstDataRow), fmt.Sprintf("A%d", endRow), firstColumnStyle)
	f.SetCellStyle(sheetName, fmt.Sprintf("B%d", FirstDataRow), fmt.Sprintf("B%d", endRow), nameStyle)
	f.SetCellStyle(sheetName, fmt.Sprintf("C%d", FirstDataRow), fmt.Sprintf("C%d", endRow), urlStyle)

	err = autoFitColumns(f, sheetName, []string{"A", "B", "C"})
	if err != nil {
		return fmt.Errorf("failed to auto-fit columns: %w", err)
	}

	err = setFreezePanes(f, sheetName, FreezeFirstColumn, FreezeFirstRow, TopLeftCellOneColumn, ActivePaneBottomRight)
	if err != nil {
		return fmt.Errorf("failed to set freeze panes: %w", err)
	}

	return nil
}

func formatScore(f *excelize.File, sheetName string, problemCount, userCount int) error {
	userNamesStyle, err := CreateHeaderStyle(f)
	if err != nil {
		return fmt.Errorf("failed to create user names style: %w", err)
	}

	totalsStyle, err := CreateTotalsStyle(f)
	if err != nil {
		return fmt.Errorf("failed to create totals style: %w", err)
	}

	endCol := userCount + ScoresSheetOffsetColumns
	endColLetter := columnIndexToLetter(endCol - 1)

	// Apply styles to user names header and totals row
	f.SetCellStyle(sheetName, fmt.Sprintf("B%d", ScoresHeaderRow), endColLetter+fmt.Sprintf("%d", ScoresHeaderRow), userNamesStyle)
	f.SetCellStyle(sheetName, fmt.Sprintf("A%d", ScoresTotalsRow), endColLetter+fmt.Sprintf("%d", ScoresTotalsRow), totalsStyle)

	if problemCount > 0 {
		problemNamesStyle, err := CreatePlainLeftAlignedStyle(f)
		if err != nil {
			return fmt.Errorf("failed to create problem names style: %w", err)
		}

		acStyle, err := CreateACStyle(f)
		if err != nil {
			return fmt.Errorf("failed to create AC style: %w", err)
		}

		endRow := problemCount + ScoresSheetOffsetRows

		// Apply formatting to problem names (column A from row 3 onwards)
		f.SetCellStyle(sheetName, fmt.Sprintf("A%d", ScoresFirstDataRow), fmt.Sprintf("A%d", endRow), problemNamesStyle)

		// Apply formatting to AC cells (from B3 onwards)
		for row := ScoresFirstDataRow; row <= endRow; row++ {
			for col := 1; col < endCol; col++ { // Start from column B (index 1)
				cellAddr := fmt.Sprintf("%c%d", FirstColumn+col, row)
				cellValue, err := f.GetCellValue(sheetName, cellAddr)
				if err != nil {
					continue
				}

				if cellValue == StatusAccepted {
					f.SetCellStyle(sheetName, cellAddr, cellAddr, acStyle)
				}
			}
		}
	}

	// Auto-fit all columns based on content
	var columns []string
	for i := 0; i < endCol; i++ {
		columns = append(columns, columnIndexToLetter(i))
	}

	err = autoFitColumns(f, sheetName, columns)
	if err != nil {
		return fmt.Errorf("failed to auto-fit columns: %w", err)
	}

	// Freeze panes - freeze first two rows and first column
	err = setFreezePanes(f, sheetName, FreezeFirstColumn, FreezeTwoRows, TopLeftCellTwoRows, ActivePaneBottomRight)
	if err != nil {
		return fmt.Errorf("failed to set freeze panes: %w", err)
	}

	return nil
}
