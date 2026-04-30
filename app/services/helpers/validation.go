package helpers

import (
	"errors"
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
)

var (
	dataNotFoundErr = &dataerr.NotFoundError{}
)

// ExistsProblem checks if a problem exists by its ID
func ExistsProblem(datastore data.DataStore, problemID int64) (bool, error) {
	if _, err := datastore.GetProblemByID(problemID); err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
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
		case errors.As(err, &dataNotFoundErr):
			return false, nil
		default:
			return false, serviceerr.MakeGeneralError(err)
		}
	}
	return true, nil
}

// IsUserIDEnrolledByClassCode checks if a user is enrolled in a class by class code
func IsUserIDEnrolledByClassCode(datastore data.DataStore, userID, classCode string) (bool, error) {
	class, err := datastore.GetClassByClassCode(classCode)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return false, serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("classCode %s doesn't match with any class", classCode), err)
		default:
			return false, serviceerr.MakeGeneralError(err)
		}
	}

	return IsUserIDEnrolledByClassID(datastore, userID, class.ID)
}

// IsUserIDEnrolledByClassID checks if a user is enrolled in a class by class ID
func IsUserIDEnrolledByClassID(datastore data.DataStore, userID string, classID int64) (bool, error) {
	isMyBoyEnrolled, err := datastore.IsUserRegisteredInClass(userID, classID)
	if err != nil {
		return false, serviceerr.NewInternalError(fmt.Sprintf("error checking if user %s is enrolled in class %d", userID, classID), err)
	}

	return isMyBoyEnrolled, nil
}
