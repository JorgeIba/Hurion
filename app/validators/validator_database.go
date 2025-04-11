package validators

import (
	"errors"
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/validatorerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
)

type ValidatorDatastore interface {
	GetClassIDByClassCode(classCode string) (int64, error)
	IsUserRegisteredInClass(userID string, classID int64) (bool, error)
	GetProblemByID(problemID int64) (*data.Problem, error)
}

type ValidatorDB struct {
	db ValidatorDatastore
}

func NewValidatorDB(d ValidatorDatastore) *ValidatorDB {
	return &ValidatorDB{db: d}
}

var (
	notFoundErr = &dataerr.NotFoundError{}
	queryErr    = &dataerr.QueryError{}
)

func (validator *ValidatorDB) IsEnrolledByClassCode(userID, classCode string) (bool, error) {
	classID, err := validator.db.GetClassIDByClassCode(classCode)
	if err != nil {
		switch {
		case errors.As(err, &notFoundErr):
			return false, validatorerr.NewNotFoundError("classID doesn't match with any class", err)
		case errors.As(err, &queryErr):
			return false, validatorerr.NewInternalError("error in DB", err)
		default:
			return false, validatorerr.NewInternalError("unexpected error", err)
		}
	}

	return validator.IsEnrolledByClassID(userID, classID)
}

func (validator *ValidatorDB) IsEnrolledByClassID(userID string, classID int64) (bool, error) {
	isMyBoyEnrolled, err := validator.db.IsUserRegisteredInClass(userID, classID)

	if err != nil {
		switch {
		case errors.As(err, &queryErr):
			return false, validatorerr.NewInternalError(fmt.Sprintf("error checking if user %s is enrolled in class %d", userID, classID), err)
		default:
			return false, validatorerr.NewInternalError("unexpected error", err)
		}
	}

	return isMyBoyEnrolled, nil
}
