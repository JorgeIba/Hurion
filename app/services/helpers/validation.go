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
			return false, serviceerr.NewClassCodeNotFoundError("classID doesn't match with any class", err)
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

// CanBroACTheProblem checks if a user can AC (solve) a problem
func CanBroACTheProblem(datastore data.DataStore, userID string, problemID int64) (bool, error) {
	problem, err := datastore.GetProblemByID(problemID)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return false, serviceerr.NewProblemNotFoundError(fmt.Sprintf("problemID %d doesn't match with any problem", problemID), err)
		default:
			return false, serviceerr.MakeGeneralError(err)
		}
	}

	if isMyBoyEnrolled, err := IsUserIDEnrolledByClassID(datastore, userID, problem.ClassID); err != nil {
		return false, err
	} else if !isMyBoyEnrolled {
		return false, serviceerr.NewUserNotEnrolledError(fmt.Sprintf("user %s is not registered in the class of problem %d", userID, problemID), nil)
	}

	if hasUserSolvedAlready, err := datastore.HasUserSolvedProblem(userID, problemID); err != nil {
		return false, serviceerr.MakeGeneralError(fmt.Errorf("error checking if user %s has solved problem %d: %v", userID, problemID, err))
	} else if hasUserSolvedAlready {
		return false, serviceerr.NewConflictError(fmt.Sprintf("user %s has already solved problem %d", userID, problemID), nil)
	}

	return true, nil
}

func HasUserSolvedProblem(datastore data.DataStore, userID string, problemID int64) (bool, error) {
	if hasUserSolvedAlready, err := datastore.HasUserSolvedProblem(userID, problemID); err != nil {
		return false, serviceerr.MakeGeneralError(fmt.Errorf("error checking if user %s has solved problem %d: %v", userID, problemID, err))
	} else if hasUserSolvedAlready {
		return true, nil
	}
	return false, nil
}
