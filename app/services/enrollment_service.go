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

// RegisterUserIntoClass registers a user (by User struct) into a class
func (service *EnrollmentService) RegisterUserIntoClass(user service_model.User, classCode string) error {
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

	if err := service.Datastore.EnsureUser(mapServiceUserToDataUser(user)); err != nil {
		return serviceerr.MakeGeneralError(err)
	}

	// Check if user was already registered
	isAlreadyRegistered, err := service.Datastore.IsUserRegisteredInClass(user.UserID, class.ID)
	if err != nil {
		return serviceerr.MakeGeneralError(err)
	} else if isAlreadyRegistered {
		return serviceerr.NewConflictError(fmt.Sprintf("user %s has already registered to this class", user.UserID), nil)
	}

	if err = service.Datastore.RegisterUserIntoClass(user.UserID, class.ID); err != nil {
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

// UserWithRolesHasAccessClass checks if at least one user role is allowed in the class.
func (service *EnrollmentService) UserWithRolesHasAccessClass(roles []string, class service_model.Class) (bool, error) {
	if class.ClassCode == "" {
		return false, serviceerr.NewClassCodeNotFoundError("class code is required to validate access", nil)
	}

	accessRoles, err := service.Datastore.GetClassAccessRolesByClassCode(class.ClassCode)
	if err != nil {
		return false, serviceerr.MakeGeneralError(err)
	}

	// Closed-by-default: classes with no configured roles deny access.
	if len(accessRoles) == 0 {
		return false, nil
	}

	roleSet := make(map[string]bool, len(roles))
	for _, roleID := range roles {
		roleSet[roleID] = true
	}

	for _, accessRole := range accessRoles {
		if roleSet[accessRole.RoleID] {
			return true, nil
		}
	}

	return false, nil
}
