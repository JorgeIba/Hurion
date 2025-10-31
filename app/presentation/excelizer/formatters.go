package excelizer

import (
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/presentation"
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

func formatProblemsList(f *excelize.File, sheetName string, problemsByTopic []ProblemsByTopic) error {
	// Apply header style to first row (A1:D1)
	headerStyle, err := CreateHeaderStyle(f)
	if err != nil {
		return fmt.Errorf("failed to create header style: %w", err)
	}
	f.SetCellStyle(sheetName, "A1", "D1", headerStyle)

	// Styles for topic header, ID, Name, URL, Difficulty columns
	topicHeaderStyle, err := CreateTopicHeaderStyle(f)
	if err != nil {
		return fmt.Errorf("failed to create topic header style: %w", err)
	}

	idStyle, err := CreatePlainCenteredStyle(f)
	if err != nil {
		return fmt.Errorf("failed to create ID style: %w", err)
	}

	nameStyle, err := CreatePlainLeftAlignedStyle(f)
	if err != nil {
		return fmt.Errorf("failed to create name style: %w", err)
	}

	urlStyle, err := CreateHyperlinkStyle(f)
	if err != nil {
		return fmt.Errorf("failed to create URL style: %w", err)
	}

	currentRow := FirstDataRow

	// Iterate through topics
	for _, topicGroup := range problemsByTopic {
		// Apply topic header style
		f.SetCellStyle(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("D%d", currentRow), topicHeaderStyle)
		currentRow++

		// Apply styles to problems in this topic
		for _, problem := range topicGroup.Problems {
			// ID (centered)
			f.SetCellStyle(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("A%d", currentRow), idStyle)

			// Name (left aligned)
			f.SetCellStyle(sheetName, fmt.Sprintf("B%d", currentRow), fmt.Sprintf("B%d", currentRow), nameStyle)

			// URL (hyperlink style)
			f.SetCellStyle(sheetName, fmt.Sprintf("C%d", currentRow), fmt.Sprintf("C%d", currentRow), urlStyle)

			// Difficulty (colored background)
			difficultyStyle, err := CreateDifficultyStyle(f, problem.Difficulty)
			if err != nil {
				return fmt.Errorf("failed to create difficulty style: %w", err)
			}
			f.SetCellStyle(sheetName, fmt.Sprintf("D%d", currentRow), fmt.Sprintf("D%d", currentRow), difficultyStyle)

			currentRow++
		}
	}

	// Auto-fit columns
	err = autoFitColumns(f, sheetName, []string{"A", "B", "C", "D"})
	if err != nil {
		return fmt.Errorf("failed to auto-fit columns: %w", err)
	}

	// Freeze first row (headers)
	err = setFreezePanes(f, sheetName, NoFreeze, FreezeFirstRow, TopLeftCellOneRow, ActivePaneBottom)
	if err != nil {
		return fmt.Errorf("failed to set freeze panes: %w", err)
	}

	return nil
}

// formatScoresSheet applies styling to the scores sheet based on metadata
func formatScoresSheet(f *excelize.File, sheetName string, tableScore presentation.TableScore) error {
	// Style for headers
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#D3D3D3"}, Pattern: 1},
	})
	if err != nil {
		return err
	}

	// Style for topic headers
	topicStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#B4C8F0"}, Pattern: 1},
	})
	if err != nil {
		return err
	}

	// Style for AC cells
	acStyle, err := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#B4E7CE"}, Pattern: 1},
	})
	if err != nil {
		return err
	}

	// Apply header style to first row
	for colIdx := range tableScore.Header {
		col := FirstColumn + colIdx
		cell := fmt.Sprintf("%c%d", col, 1)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	// Apply styles to data rows based on metadata
	for rowIdx, metadata := range tableScore.RowMetadata {
		excelRow := rowIdx + 2

		if metadata.IsTopicHeader {
			// Apply topic header style to entire row
			for colIdx := range tableScore.Table[rowIdx] {
				col := FirstColumn + colIdx
				cell := fmt.Sprintf("%c%d", col, excelRow)
				f.SetCellStyle(sheetName, cell, cell, topicStyle)
			}
		} else {
			// Apply cell-specific styles based on cell metadata
			for colIdx, cellMeta := range metadata.CellMetadata {
				if cellMeta.IsAC {
					col := FirstColumn + colIdx
					cell := fmt.Sprintf("%c%d", col, excelRow)
					f.SetCellStyle(sheetName, cell, cell, acStyle)
				}
			}
		}
	}

	return nil
}
