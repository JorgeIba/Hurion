package services

import (
	"errors"
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
)

var (
	dataNotFoundErrUser = &dataerr.NotFoundError{}
)

// UserService handles user management operations
type UserService struct {
	Datastore data.DataStore
}

// EnsureUser ensures that a user with the given ID and display name exists in the database
func (service *UserService) EnsureUser(user service_model.User) error {
	err := service.Datastore.EnsureUser(mapServiceUserToDataUser(user))
	if err != nil {
		return serviceerr.MakeGeneralError(err)
	}

	return nil
}

// ExterminateUser removes all data for a userID
func (service *UserService) ExterminateUser(userID string) error {
	// First check if user exists
	if _, err := service.GetUserByUserID(userID); err != nil {
		return err
	}

	if err := service.Datastore.ExterminateUser(userID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

// GetUserByUserID retrieves a user by their ID
func (service *UserService) GetUserByUserID(userID string) (service_model.User, error) {
	user, err := service.Datastore.GetUserByUserID(userID)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErrUser):
			return service_model.User{}, serviceerr.NewUserNotFoundError(fmt.Sprintf("userID %s not found", userID), err)
		default:
			return service_model.User{}, serviceerr.MakeGeneralError(err)
		}
	}
	return mapDataUserToServiceUser(user), nil
}

// GetClassesByUserID retrieves all classes that a user is enrolled in
func (service *UserService) GetClassesByUserID(userID string) ([]service_model.Class, error) {
	classes, err := service.Datastore.GetClassesByUserID(userID)
	if err != nil {
		return nil, serviceerr.MakeGeneralError(fmt.Errorf("something happened trying to get classes by userID: %s: %v", userID, err))
	}
	return mapDataClassesToServiceClasses(classes), nil
}
