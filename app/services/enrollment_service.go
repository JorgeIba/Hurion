package services

import (
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
)

// EnrollmentService handles user enrollment into classes
type EnrollmentService struct {
	Datastore data.DataStore
}

// GetListOfUsersByClassCode retrieves all active users in a class (users with at least one solve)
func (service *EnrollmentService) GetListOfUsersByClassCode(classCode string) ([]service_model.User, error) {
	class, err := service.Datastore.GetClassByClassCode(classCode)
	if err != nil {
		switch {
		case dataerr.IsNotFound(err):
			return []service_model.User{}, serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("classCode %s doesn't match with any classID", classCode), err)
		default:
			return []service_model.User{}, serviceerr.MakeGeneralError(err)
		}
	}

	users, err := service.Datastore.GetActiveUsersByClassID(class.ID)
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
