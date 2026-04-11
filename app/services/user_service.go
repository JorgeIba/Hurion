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

// BindUser binds a Discord userID to a platform username
func (service *UserService) BindUser(userID, userName string) error {
	isUsernameAvailable, err := service.Datastore.IsUsernameAvailable(userName, userID)
	if err != nil {
		return serviceerr.MakeGeneralError(err)
	} else if !isUsernameAvailable {
		return serviceerr.NewConflictError(fmt.Sprintf("username `%s` is already taken", userName), nil)
	}

	err = service.Datastore.BindUser(data.User{UserID: userID, UserName: userName})
	if err != nil {
		return serviceerr.MakeGeneralError(err)
	}

	return nil
}

// ChangeUsername changes an existing username for a userID
func (service *UserService) ChangeUsername(userID, newUsername string) error {
	// Logic is actually the same as BindUser due to SQL implementation (upsert or replace logic if exists)
	// But we keep it separate for semantic reasons
	return service.BindUser(userID, newUsername)
}

// HasUsernameBound checks if a userID has a username bound
func (service *UserService) HasUsernameBound(userID string) (bool, error) {
	hasBound, err := service.Datastore.HasUsernameBound(userID)
	if err != nil {
		return false, serviceerr.MakeGeneralError(err)
	}
	return hasBound, nil
}

// ExterminateUser removes all data for a userID
func (service *UserService) ExterminateUser(userID string) error {
	// First check if user exists
	if _, err := service.Datastore.GetUserByUserID(userID); err != nil {
		switch {
		case errors.As(err, &dataNotFoundErrUser):
			return serviceerr.NewUserNotFoundError(fmt.Sprintf("userID %s not found", userID), err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	if err := service.Datastore.ExterminateUser(userID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

// GetUsernameByUserID retrieves the username for a userID
func (service *UserService) GetUsernameByUserID(userID string) (string, error) {
	user, err := service.Datastore.GetUserByUserID(userID)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErrUser):
			return "", serviceerr.NewUserNotFoundError(fmt.Sprintf("userID %s not found", userID), err)
		default:
			return "", serviceerr.MakeGeneralError(err)
		}
	}
	return user.UserName, nil
}

// GetUserIDByUsername retrieves the userID for a username
func (service *UserService) GetUserIDByUsername(userName string) (string, error) {
	user, err := service.Datastore.GetUserByUserName(userName)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErrUser):
			return "", serviceerr.NewUserNotFoundError(fmt.Sprintf("userName %s not found", userName), err)
		default:
			return "", serviceerr.MakeGeneralError(err)
		}
	}
	return user.UserID, nil
}

// GetClassesByUserID retrieves all classes that a user is enrolled in
func (service *UserService) GetClassesByUserID(userID string) ([]service_model.Class, error) {
	classes, err := service.Datastore.GetClassesByUserID(userID)
	if err != nil {
		return nil, serviceerr.MakeGeneralError(fmt.Errorf("something happened trying to get classes by userID: %s: %v", userID, err))
	}
	return mapDataClassesToServiceClasses(classes), nil
}
