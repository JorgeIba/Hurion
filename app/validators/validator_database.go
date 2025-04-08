package validators

import (
	"fmt"

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

func (validator *ValidatorDB) IsEnrolledByClassCode(userID, classCode string) (bool, error) {
	classID, err := validator.db.GetClassIDByClassCode(classCode)
	if err != nil {
		return false, fmt.Errorf("could not get the class reference, are you sure it exists?: %v", err)
	}

	return validator.IsEnrolledByClassID(userID, classID)
}

func (validator *ValidatorDB) ExistsProblem(problemID int64) (bool, error) {
	_, err := validator.db.GetProblemByID(problemID)
	if err != nil {
		return false, fmt.Errorf("could not check problem info")
	}

	return true, nil
}

func (validator *ValidatorDB) ExistsProblemInClass(problemID, classID int64) (bool, error) {
	problem, err := validator.db.GetProblemByID(problemID)
	if err != nil {
		return false, fmt.Errorf("could not check problem info")
	}
	return problem.ClassID == classID, nil
}

func (validator *ValidatorDB) IsEnrolledByClassID(userID string, classID int64) (bool, error) {
	isMyBoyEnrolled, err := validator.db.IsUserRegisteredInClass(userID, classID)
	if err != nil {
		return false, fmt.Errorf("error checking if user %s is enrolled in class %d: %v", userID, classID, err)
	}

	return isMyBoyEnrolled, nil
}
