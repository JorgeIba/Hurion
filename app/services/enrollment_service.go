package services

import (
	"errors"
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/helpers"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
)

var (
	dataNotFoundErrEnrollment = &dataerr.NotFoundError{}
)

// EnrollmentService handles user enrollment into classes
type EnrollmentService struct {
	Datastore data.DataStore
}

// RegisterUserIDIntoClass registers a user (by userID) into a class
func (service *EnrollmentService) RegisterUserIDIntoClass(userID, classCode string) error {
	// Get class
	class, err := service.Datastore.GetClassByClassCode(classCode)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErrEnrollment):
			return serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("classCode %s doesn't match with any classID", classCode), err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	// Discord User should have binded a User Name
	isUserBinded, err := service.Datastore.IsUserBinded(userID)
	if err != nil {
		return serviceerr.MakeGeneralError(err)
	} else if !isUserBinded {
		return serviceerr.NewUserNotBindedError(fmt.Sprintf("user %s has not username binded yet", userID), nil)
	}

	// Check if user was already registered
	isAlreadyRegistered, err := service.Datastore.IsUserRegisteredInClass(userID, class.ID)
	if err != nil {
		return serviceerr.MakeGeneralError(err)
	} else if isAlreadyRegistered {
		return serviceerr.NewConflictError(fmt.Sprintf("user %s has already registered to this class", userID), nil)
	}

	if err = service.Datastore.RegisterUserIntoClass(userID, class.ID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

// DeregisterUserIDFromClass removes a user (by userID) from a class
func (service *EnrollmentService) DeregisterUserIDFromClass(userID string, classCode string) error {
	// Validate if user is registered in the class
	isMyBoyEnrolled, err := helpers.IsUserIDEnrolledByClassCode(service.Datastore, userID, classCode)
	if err != nil {
		return err
	} else if !isMyBoyEnrolled {
		return serviceerr.NewUserNotEnrolledError(fmt.Sprintf("user %s is not registered in class %s", userID, classCode), nil)
	}

	// Get Info
	class, err := service.Datastore.GetClassByClassCode(classCode)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErrEnrollment):
			return serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("classCode %s doesn't match with any classID", classCode), err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	// Deregister
	if err = service.Datastore.DeregisterUserFromClass(userID, class.ID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

// RegisterUsernameIntoClass registers a user (by username) into a class
func (service *EnrollmentService) RegisterUsernameIntoClass(userName, classCode string) error {
	user, err := service.Datastore.GetUserByUserName(userName)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErrEnrollment):
			return serviceerr.NewUserNotFoundError(fmt.Sprintf("username %s not found", userName), err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	err = service.RegisterUserIDIntoClass(user.UserID, classCode)
	return err
}

// DeregisterUsernameFromClass removes a user (by username) from a class.
func (service *EnrollmentService) DeregisterUsernameFromClass(userName, classCode string) error {
	user, err := service.Datastore.GetUserByUserName(userName)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErrEnrollment):
			return serviceerr.NewUserNotFoundError(fmt.Sprintf("username %s not found", userName), err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	return service.DeregisterUserIDFromClass(user.UserID, classCode)
}

// GetListOfUsersByClassCode retrieves all users enrolled in a class
func (service *EnrollmentService) GetListOfUsersByClassCode(classCode string) ([]service_model.User, error) {
	class, err := service.Datastore.GetClassByClassCode(classCode)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErrEnrollment):
			return []service_model.User{}, serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("classCode %s doesn't match with any classID", classCode), err)
		default:
			return []service_model.User{}, serviceerr.MakeGeneralError(err)
		}
	}

	users, err := service.Datastore.GetAllUsersByClassID(class.ID)
	if err != nil {
		return []service_model.User{}, serviceerr.MakeGeneralError(err)
	}

	return mapDataUsersToServiceUsers(users), nil
}
