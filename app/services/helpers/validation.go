package helpers

import (
	"fmt"

	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/services/service_model"
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

// ValidateBatchSize checks the shared operation limit before reading or writing.
func ValidateBatchSize(size int) error {
	if size < 1 || size > service_model.MaxBatchOperations {
		return serviceerr.NewValidationError(fmt.Sprintf("batch must contain between 1 and %d operations", service_model.MaxBatchOperations), nil)
	}
	return nil
}
