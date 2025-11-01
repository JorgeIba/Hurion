package excelizer

import (
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/presentationerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/presentation"
	"github.com/xuri/excelize/v2"
)

func CreateExcel() *excelize.File {
	f := excelize.NewFile()
	return f
}

// Generates an Excel file with class data
// Takes users, problems grouped by topic, and a TableScore for rendering
// Returns file bytes, filename, and error
func ExportClassToExcel(users []User, problemsByTopic []ProblemsByTopic, tableScore presentation.TableScore, classCode string) ([]byte, string, error) {
	// Sort data
	SortUsersAlphabetically(users)

	// Create Excel file
	excel := CreateExcel()
	defer excel.Close()

	if err := AddUsersSheet(excel, "Users", users); err != nil {
		return nil, "", presentationerr.NewDataWritingError("error adding users sheet", err)
	}

	// Pass problemsByTopic directly - AddProblemsSheet handles the structure
	if err := AddProblemsSheet(excel, "Problems", problemsByTopic); err != nil {
		return nil, "", presentationerr.NewDataWritingError("error adding problems sheet", err)
	}

	// Use TableScore to render scores sheet with metadata
	if err := AddScoresSheetFromTableScore(excel, "Scores", tableScore); err != nil {
		return nil, "", presentationerr.NewDataWritingError("error adding scores sheet", err)
	}

	// Remove default sheet
	excel.DeleteSheet("Sheet1")

	// Convert to bytes
	buffer, err := excel.WriteToBuffer()
	if err != nil {
		return nil, "", presentationerr.NewFormatError("error writing excel to buffer", err)
	}

	filename := classCode + "_export.xlsx"
	return buffer.Bytes(), filename, nil
}
