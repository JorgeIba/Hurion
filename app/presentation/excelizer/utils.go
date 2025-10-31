package excelizer

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// columnLetterToIndex converts Excel column letters to 0-based index
// Examples: "A" -> 0, "B" -> 1, "Z" -> 25, "AA" -> 26, "AB" -> 27
func columnLetterToIndex(col string) int {
	col = strings.ToUpper(col)
	index, base := 0, 26
	for i := 0; i < len(col); i++ {
		index = index*base + int(col[i]-'A') + 1
	}
	return index - 1
}

// columnIndexToLetter converts 0-based index to Excel column letters
// Examples: 0 -> "A", 1 -> "B", 25 -> "Z", 26 -> "AA", 27 -> "AB"
func columnIndexToLetter(index int) string {
	result := ""
	index++
	for index > 0 {
		index--
		result = string(rune('A'+(index%26))) + result
		index /= 26
	}
	return result
}

// autoFitColumns automatically adjusts column width based on content
func autoFitColumns(f *excelize.File, sheetName string, columns []string) error {
	for _, col := range columns {
		maxWidth := MinColumnWidth

		// Get all rows in the sheet
		rows, err := f.GetRows(sheetName)
		if err != nil {
			return fmt.Errorf("failed to get rows: %w", err)
		}

		colIndex := columnLetterToIndex(col)

		// Check each row for this column
		for _, row := range rows {
			if colIndex < len(row) {
				content := row[colIndex]
				maxWidth = max(maxWidth, float64(len(content))*ColumnWidthMultiplier)
			}
		}

		// Add buffer to the width and cap the maximum width
		maxWidth = min(maxWidth+ColumnWidthBuffer, MaxColumnWidth)

		f.SetColWidth(sheetName, col, col, maxWidth)
	}

	return nil
}

// setFreezePanes is a helper to set freeze panes on a sheet
func setFreezePanes(f *excelize.File, sheetName string, xSplit, ySplit int, topLeftCell, activePane string) error {
	return f.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      xSplit,
		YSplit:      ySplit,
		TopLeftCell: topLeftCell,
		ActivePane:  activePane,
	})
}
