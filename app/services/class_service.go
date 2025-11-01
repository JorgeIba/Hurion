package services

import (
	"errors"
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
)

var (
	dataNotFoundErrClass = &dataerr.NotFoundError{}
	dataConflictErrClass = &dataerr.RowConflictError{}
)

// ClassService handles class management operations
type ClassService struct {
	Datastore data.DataStore
}

// CreateClass creates a new class
func (service *ClassService) CreateClass(classCode, className string) error {
	_, err := service.Datastore.CreateClass(data.Class{
		ClassCode: classCode,
		Name:      className,
	})

	if err != nil {
		switch {
		case errors.As(err, &dataConflictErrClass):
			return serviceerr.NewConflictError("", err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	return nil
}

// DeleteClassByClassCode deletes a class by its class code
func (service *ClassService) DeleteClassByClassCode(classCode string) error {
	class, err := service.Datastore.GetClassByClassCode(classCode)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErrClass):
			return serviceerr.NewNotFoundInfoError("", err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	if err = service.Datastore.ClearClass(class.ID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

// GetAllClasses retrieves all classes
func (service *ClassService) GetAllClasses() ([]data.Class, error) {
	classes, err := service.Datastore.GetAllClasses()
	if err != nil {
		return []data.Class{}, serviceerr.MakeGeneralError(err)
	}
	return classes, nil
}

// GetClassIDByClassCode retrieves the class ID for a given class code
func (service *ClassService) GetClassIDByClassCode(classCode string) (int64, error) {
	class, err := service.Datastore.GetClassByClassCode(classCode)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErrClass):
			return 0, serviceerr.NewNotFoundInfoError(fmt.Sprintf("classCode %s doesn't match with any classID", classCode), err)
		default:
			return 0, serviceerr.MakeGeneralError(err)
		}
	}

	return class.ID, nil
}
