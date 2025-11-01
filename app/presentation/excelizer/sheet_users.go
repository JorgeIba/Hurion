package excelizer

import (
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/presentationerr"
	"github.com/xuri/excelize/v2"
)

func writeUsersData(f *excelize.File, sheetName string, users []User) error {
	headers := []string{"User ID", "User Name"}
	for i, header := range headers {
		cell := makeCellRef(i, HeaderRow)
		f.SetCellValue(sheetName, cell, header)
	}

	for i, user := range users {
		row := i + FirstDataRow

		// User ID column (A)
		cellUserID := makeCellRef(0, row)
		f.SetCellValue(sheetName, cellUserID, user.ID)

		// User Name column (B)
		cellUserName := makeCellRef(1, row)
		f.SetCellValue(sheetName, cellUserName, user.Name)
	}

	return nil
}

// AddUsersSheet adds a new sheet with a list of users
func AddUsersSheet(f *excelize.File, sheetName string, users []User) error {
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return presentationerr.NewDataWritingError("failed to create sheet "+sheetName, err)
	}

	f.SetActiveSheet(index)

	err = writeUsersData(f, sheetName, users)
	if err != nil {
		return presentationerr.NewDataWritingError("failed to write users data", err)
	}

	return formatUsersList(f, sheetName, len(users))
}
