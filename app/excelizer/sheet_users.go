package excelizer

import (
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/xuri/excelize/v2"
)

func writeUsersData(f *excelize.File, sheetName string, users []data.User) error {
	// Write headers
	headers := []string{"User ID", "User Name"}
	for i, header := range headers {
		cell := fmt.Sprintf("%c%d", FirstColumn+i, HeaderRow)
		f.SetCellValue(sheetName, cell, header)
	}

	// Write user data
	for i, user := range users {
		row := i + FirstDataRow

		// User ID column (A)
		cellUserID := fmt.Sprintf("A%d", row)
		f.SetCellValue(sheetName, cellUserID, user.UserID)

		// User Name column (B)
		cellUserName := fmt.Sprintf("B%d", row)
		f.SetCellValue(sheetName, cellUserName, user.UserName)
	}

	return nil
}

// AddUsersSheet adds a new sheet with a list of users
func AddUsersSheet(f *excelize.File, sheetName string, users []data.User) error {
	// Create new sheet
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("failed to create sheet %s: %w", sheetName, err)
	}

	// Set the sheet as active
	f.SetActiveSheet(index)

	// Write data
	err = writeUsersData(f, sheetName, users)
	if err != nil {
		return fmt.Errorf("failed to write users data: %w", err)
	}

	// Apply formatting
	return formatUsersList(f, sheetName, len(users))
}
