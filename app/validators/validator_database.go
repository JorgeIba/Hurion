package validators

import (
	"fmt"
)

type ValidatorDatastore interface {
	GetClassIDByClassCode(classCode string) (int64, error)
	IsUserRegisteredInClass(userID string, classID int64) (bool, error)
}

type ValidatorDB struct {
	db ValidatorDatastore
}

func NewValidatorDB(d ValidatorDatastore) *ValidatorDB {
	return &ValidatorDB{db: d}
}

func (validator *ValidatorDB) IsEnrolled(userID, classCode string) (bool, error) {
	classID, err := validator.db.GetClassIDByClassCode(classCode)
	if err != nil {
		return false, fmt.Errorf("could not get the class reference, are you sure it exists?: %v", err)
	}

	isMyBoyEnrolled, err := validator.db.IsUserRegisteredInClass(userID, classID)
	if err != nil {
		return false, fmt.Errorf("error checking if user %s is enrolled in class %d: %v", userID, classID, err)
	}

	return isMyBoyEnrolled, nil
}
