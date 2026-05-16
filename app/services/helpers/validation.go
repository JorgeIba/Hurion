package helpers

import (
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
)

// ExistsProblem checks if a problem exists by its ID
func ExistsProblem(datastore data.DataStore, problemID int64) (bool, error) {
	if _, err := datastore.GetProblemByID(problemID); err != nil {
		switch {
		case dataerr.IsNotFound(err):
			return false, nil
		default:
			return false, serviceerr.MakeGeneralError(err)
		}
	}
	return true, nil
}

// IsClassCodeValid checks if a class code exists
func IsClassCodeValid(datastore data.DataStore, classCode string) (bool, error) {
	if _, err := datastore.GetClassByClassCode(classCode); err != nil {
		switch {
		case dataerr.IsNotFound(err):
			return false, nil
		default:
			return false, serviceerr.MakeGeneralError(err)
		}
	}
	return true, nil
}
