package excelizer

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// CreateExcel creates a new Excel file and returns the excelize.File pointer
func CreateExcel() *excelize.File {
	f := excelize.NewFile()
	return f
}

// SaveExcel saves the Excel file to the specified path
func SaveExcel(f *excelize.File, filePath string) error {
	if err := f.SaveAs(filePath); err != nil {
		return fmt.Errorf("failed to save Excel file: %w", err)
	}
	return nil
}
