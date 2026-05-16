package services

import (
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
)

type ClassService struct {
	Datastore data.DataStore
}

func NewClassService(ds data.DataStore) *ClassService {
	return &ClassService{
		Datastore: ds,
	}
}

// CreateClass creates a new class
func (service *ClassService) CreateClass(classCode, className string) (int64, error) {
	classID, err := service.Datastore.CreateClass(data.Class{ClassCode: classCode, Name: className})

	if err != nil {
		switch {
		case dataerr.IsRowConflict(err):
			return 0, serviceerr.NewConflictError(fmt.Sprintf("class with code %s already exists", classCode), err)
		default:
			return 0, serviceerr.MakeGeneralError(err)
		}
	}

	return classID, nil
}

// DeleteClassByClassCode deletes a class by its class code
func (service *ClassService) DeleteClassByClassCode(classCode string) error {
	class, err := service.Datastore.GetClassByClassCode(classCode)
	if err != nil {
		switch {
		case dataerr.IsNotFound(err):
			return serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("class %s not found", classCode), err)
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
func (service *ClassService) GetAllClasses() ([]service_model.Class, error) {
	classes, err := service.Datastore.GetAllClasses()
	if err != nil {
		return nil, serviceerr.MakeGeneralError(err)
	}
	return mapDataClassesToServiceClasses(classes), nil
}

// GetClassByClassCode returns a class entity by its unique class code.
func (service *ClassService) GetClassByClassCode(classCode string) (service_model.Class, error) {
	class, err := service.Datastore.GetClassByClassCode(classCode)

	if err != nil {
		switch {
		case dataerr.IsNotFound(err):
			return service_model.Class{}, serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("classCode %s doesn't match with any class", classCode), err)
		default:
			return service_model.Class{}, serviceerr.MakeGeneralError(err)
		}
	}

	return mapDataClassToServiceClass(class), nil
}

// GetClassIDByClassCode retrieves the class ID for a given class code
func (service *ClassService) GetClassIDByClassCode(classCode string) (int64, error) {
	class, err := service.Datastore.GetClassByClassCode(classCode)

	if err != nil {
		switch {
		case dataerr.IsNotFound(err):
			return 0, serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("classCode %s doesn't match with any classID", classCode), err)
		default:
			return 0, serviceerr.MakeGeneralError(err)
		}
	}

	return class.ID, nil
}

// GetClassByClassID returns a class entity by its unique numeric ID.
func (service *ClassService) GetClassByClassID(classID int64) (service_model.Class, error) {
	class, err := service.Datastore.GetClassByClassID(classID)

	if err != nil {
		switch {
		case dataerr.IsNotFound(err):
			return service_model.Class{}, serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("classID %d doesn't match with any class", classID), err)
		default:
			return service_model.Class{}, serviceerr.MakeGeneralError(err)
		}
	}

	return mapDataClassToServiceClass(class), nil
}

// RegisterClassroom assigns a room to a class.
// It validates that the room is not already registered to a different class.
func (service *ClassService) RegisterClassroom(classroom service_model.Classroom) error {
	// Check if class exists
	if _, err := service.Datastore.GetClassByClassCode(classroom.ClassCode); err != nil {
		switch {
		case dataerr.IsNotFound(err):
			return serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("class %s not found", classroom.ClassCode), err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	// Check if room is already registered to another class
	ownerClass, exists, err := service.Datastore.IsRoomRegistered(classroom.RoomID)
	if err != nil {
		return serviceerr.MakeGeneralError(err)
	}

	if exists {
		if ownerClass != classroom.ClassCode {
			return serviceerr.NewConflictError(fmt.Sprintf("room %s is already registered to class %s", classroom.RoomID, ownerClass), nil)
		}
		// ownerClass == classroom.ClassCode
		// Room is already registered to the same class, no action needed
		return nil
	}

	// Delegate to datastore
	if err := service.Datastore.AddClassroom(mapServiceClassroomToDataClassroom(classroom)); err != nil {
		return serviceerr.MakeGeneralError(err)
	}

	return nil
}

// RemoveClassroom unlinks a room from its class.
func (service *ClassService) RemoveClassroom(roomID string) error {
	if err := service.Datastore.RemoveClassroomByRoomID(roomID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

// GetClassrooms returns all classrooms registered to a specific class.
func (service *ClassService) GetClassrooms(classCode string) ([]service_model.Classroom, error) {
	classrooms, err := service.Datastore.GetClassroomsByClassCode(classCode)
	if err != nil {
		return nil, serviceerr.MakeGeneralError(err)
	}

	return mapDataClassroomsToServiceClassrooms(classrooms), nil
}

// AssignAccessRole links a role identifier to a class for permission checks.
func (service *ClassService) AssignAccessRole(role service_model.ClassAccessRole) error {
	// Verify class exists
	if _, err := service.Datastore.GetClassByClassCode(role.ClassCode); err != nil {
		switch {
		case dataerr.IsNotFound(err):
			return serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("class %s not found", role.ClassCode), err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	if err := service.Datastore.AddClassAccessRole(mapServiceRoleToDataRole(role)); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

// RevokeAccessRole unlinks a role from a class.
func (service *ClassService) RevokeAccessRole(role service_model.ClassAccessRole) error {
	if err := service.Datastore.RemoveClassAccessRole(mapServiceRoleToDataRole(role)); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

// GetAccessRoles returns all access roles registered to a specific class.
func (service *ClassService) GetAccessRoles(classCode string) ([]service_model.ClassAccessRole, error) {
	roles, err := service.Datastore.GetClassAccessRolesByClassCode(classCode)
	if err != nil {
		return nil, serviceerr.MakeGeneralError(err)
	}

	return mapDataRolesToServiceRoles(roles), nil
}

// GetClassCodeFromRoom resolves the class code linked to a room.
// The found return value is false when room has no class mapping.
func (service *ClassService) GetClassCodeFromRoom(roomID string) (string, bool, error) {
	classCode, err := service.Datastore.GetClassCodeByRoomID(roomID)
	if err != nil {
		switch {
		case dataerr.IsNotFound(err):
			return "", false, nil
		default:
			return "", false, serviceerr.MakeGeneralError(err)
		}
	}

	return classCode, true, nil
}
