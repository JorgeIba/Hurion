package services

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

// UserService handles user-specific operations
type UserService struct {
	Datastore data.DataStore
}

// BindUser binds a Discord userID to a username
func (service *UserService) BindUser(userID, userName string) error {
	// Check if the username is available for this user
	isAvailable, err := service.Datastore.IsUsernameAvailable(userName, userID)
	if err != nil {
		return serviceerr.MakeGeneralError(err)
	}

	if !isAvailable {
		return serviceerr.NewConflictError(fmt.Sprintf("username '%s' is already taken by another user", userName), nil)
	}

	// Proceed with binding the user
	if err := service.Datastore.BindUser(data.User{
		UserID:   userID,
		UserName: userName,
	}); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

// ChangeUsername changes the username for a given userID
func (service *UserService) ChangeUsername(userID, newUserName string) error {
	// Check if the new username is available
	isAvailable, err := service.Datastore.IsUsernameAvailable(newUserName, userID)
	if err != nil {
		return serviceerr.MakeGeneralError(err)
	} else if !isAvailable {
		return serviceerr.NewConflictError(fmt.Sprintf("username '%s' is already taken by another user", newUserName), nil)
	}

	if err := service.Datastore.BindUser(data.User{
		UserID:   userID,
		UserName: newUserName,
	}); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

// HasUsernameBound checks if a userID has a username bound
func (service *UserService) HasUsernameBound(userID string) (bool, error) {
	hasBound, err := service.Datastore.HasUsernameBound(userID)
	if err != nil {
		return false, serviceerr.MakeGeneralError(err)
	}
	return hasBound, nil
}

// ExterminateUser deletes a user completely from the system
func (service *UserService) ExterminateUser(userID string) error {
	_, err := service.Datastore.GetUserByUserID(userID)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return serviceerr.NewNotFoundInfoError("", err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	if err = service.Datastore.ExterminateUser(userID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

// GetUsernameByUserID retrieves the username for a given userID
func (service *UserService) GetUsernameByUserID(userID string) (string, error) {
	user, err := service.Datastore.GetUserByUserID(userID)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return "", serviceerr.NewNotFoundInfoError(fmt.Sprintf("userID %s doesn't match with any user", userID), err)
		default:
			return "", serviceerr.MakeGeneralError(err)
		}
	}
	return user.UserName, nil
}

// GetUserIDByUsername retrieves the userID for a given username
func (service *UserService) GetUserIDByUsername(userName string) (string, error) {
	user, err := service.Datastore.GetUserByUserName(userName)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return "", serviceerr.NewNotFoundInfoError("", err)
		default:
			return "", serviceerr.MakeGeneralError(err)
		}
	}

	return user.UserID, nil
}

// GetClassesByUserID retrieves all classes that a user is enrolled in
func (service *UserService) GetClassesByUserID(userID string) ([]data.Class, error) {
	classes, err := service.Datastore.GetClassesByUserID(userID)
	if err != nil {
		return []data.Class{}, serviceerr.MakeGeneralError(fmt.Errorf("something happened trying to get classes by userID: %s: %v", userID, err))
	}
	return classes, nil
}
