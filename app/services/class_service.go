package services

import (
	"fmt"

	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

type ClassService struct {
	Datastore  data.DataStore
	Authorizer Authorizer
}

func NewClassService(ds data.DataStore, auth Authorizer) *ClassService {
	return &ClassService{
		Datastore:  ds,
		Authorizer: auth,
	}
}

// CreateClass creates a new class
func (service *ClassService) CreateClass(actor service_model.Actor, classCode, className string) (int64, error) {
	if err := validateAuthorizationInputs(service.Authorizer, actor); err != nil {
		return 0, err
	}

	if err := requirePermission(service.Authorizer.CanCreateClass(actor)); err != nil {
		return 0, err
	}

	if err := service_model.ValidateClassCodeLength(classCode); err != nil {
		return 0, serviceerr.NewValidationError("invalid class code", err)
	}

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

// ClearClassContentsByClassCode deletes all problems and score-driving data from a class while preserving the class itself.
func (service *ClassService) ClearClassContentsByClassCode(actor service_model.Actor, classCode string) error {
	if err := validateAuthorizationInputs(service.Authorizer, actor); err != nil {
		return err
	}

	if err := requirePermission(service.Authorizer.CanClearClass(actor)); err != nil {
		return err
	}

	class, err := service.Datastore.GetClassByClassCode(classCode)
	if err != nil {
		switch {
		case dataerr.IsNotFound(err):
			return serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("class %s not found", classCode), err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	if err = service.Datastore.ClearClassContents(class.ID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}

	return nil
}

// DeleteClassByClassCode deletes a class by its class code.
func (service *ClassService) DeleteClassByClassCode(actor service_model.Actor, classCode string) error {
	if err := validateAuthorizationInputs(service.Authorizer, actor); err != nil {
		return err
	}

	if err := requirePermission(service.Authorizer.CanDeleteClass(actor)); err != nil {
		return err
	}

	class, err := service.Datastore.GetClassByClassCode(classCode)
	if err != nil {
		switch {
		case dataerr.IsNotFound(err):
			return serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("class %s not found", classCode), err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	if err = service.Datastore.DeleteClass(class.ID); err != nil {
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
	if err := service_model.ValidateClassCodeLength(classCode); err != nil {
		return service_model.Class{}, serviceerr.NewValidationError("invalid class code", err)
	}

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
	if err := service_model.ValidateClassCodeLength(classCode); err != nil {
		return 0, serviceerr.NewValidationError("invalid class code", err)
	}

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

// LinkClassroom assigns a room to a class and reports whether the link was new.
func (service *ClassService) LinkClassroom(actor service_model.Actor, classroom service_model.Classroom) (bool, error) {
	results, err := service.LinkClassrooms(actor, classroom.ClassCode, classroom.RoomID)
	if err != nil {
		return false, err
	}

	return results[0].Linked, results[0].Err
}

// LinkClassrooms authorizes once and links rooms independently in input order.
// A failed room does not prevent the remaining rooms from being linked.
func (service *ClassService) LinkClassrooms(actor service_model.Actor, classCode string, roomIDs ...string) ([]service_model.ClassResourceLinkResult, error) {
	if err := service.requireClassLinking(actor, classCode); err != nil {
		return nil, err
	}

	results := make([]service_model.ClassResourceLinkResult, len(roomIDs))
	for i, roomID := range roomIDs {
		linked, err := service.linkClassroom(classCode, roomID)
		results[i] = service_model.ClassResourceLinkResult{ResourceID: roomID, Linked: linked, Err: err}
	}

	return results, nil
}

func (service *ClassService) linkClassroom(classCode, roomID string) (bool, error) {
	ownerClass, exists, err := service.Datastore.IsRoomRegistered(roomID)
	if err != nil {
		return false, serviceerr.MakeGeneralError(err)
	}

	if exists {
		if ownerClass != classCode {
			return false, serviceerr.NewConflictError(fmt.Sprintf("room %s is already registered to class %s", roomID, ownerClass), nil)
		}

		return false, nil
	}

	if err := service.Datastore.AddClassroom(data.Classroom{ClassCode: classCode, RoomID: roomID}); err != nil {
		return false, serviceerr.MakeGeneralError(err)
	}

	return true, nil
}

// RemoveClassroom unlinks a room from its class.
func (service *ClassService) RemoveClassroom(actor service_model.Actor, roomID string) error {
	if err := validateAuthorizationInputs(service.Authorizer, actor); err != nil {
		return err
	}

	if err := requirePermission(service.Authorizer.CanUnlinkClassResources(actor)); err != nil {
		return err
	}

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

// LinkClassAccessRole links a role to a class and reports whether the link was new.
func (service *ClassService) LinkClassAccessRole(actor service_model.Actor, role service_model.ClassAccessRole) (bool, error) {
	results, err := service.LinkClassAccessRoles(actor, role.ClassCode, role.RoleID)
	if err != nil {
		return false, err
	}

	return results[0].Linked, results[0].Err
}

// LinkClassAccessRoles authorizes once and links roles independently in input order.
// A failed role does not prevent the remaining roles from being linked.
func (service *ClassService) LinkClassAccessRoles(actor service_model.Actor, classCode string, roleIDs ...string) ([]service_model.ClassResourceLinkResult, error) {
	if err := service.requireClassLinking(actor, classCode); err != nil {
		return nil, err
	}

	results := make([]service_model.ClassResourceLinkResult, len(roleIDs))
	for i, roleID := range roleIDs {
		linked, err := service.linkClassAccessRole(classCode, roleID)
		results[i] = service_model.ClassResourceLinkResult{ResourceID: roleID, Linked: linked, Err: err}
	}

	return results, nil
}

func (service *ClassService) requireClassLinking(actor service_model.Actor, classCode string) error {
	if err := validateAuthorizationInputs(service.Authorizer, actor); err != nil {
		return err
	}

	if err := requirePermission(service.Authorizer.CanLinkClassResources(actor)); err != nil {
		return err
	}

	if _, err := service.Datastore.GetClassByClassCode(classCode); err != nil {
		if dataerr.IsNotFound(err) {
			return serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("class %s not found", classCode), err)
		}

		return serviceerr.MakeGeneralError(err)
	}

	return nil
}

func (service *ClassService) linkClassAccessRole(classCode, roleID string) (bool, error) {
	exists, err := service.Datastore.HasClassAccessRole(classCode, roleID)
	if err != nil {
		return false, serviceerr.MakeGeneralError(err)
	}
	if exists {
		return false, nil
	}

	if err := service.Datastore.AddClassAccessRole(data.ClassAccessRole{ClassCode: classCode, RoleID: roleID}); err != nil {
		return false, serviceerr.MakeGeneralError(err)
	}

	return true, nil
}

// RevokeAccessRole unlinks a role from a class.
func (service *ClassService) RevokeAccessRole(actor service_model.Actor, role service_model.ClassAccessRole) error {
	if err := validateAuthorizationInputs(service.Authorizer, actor); err != nil {
		return err
	}

	if err := requirePermission(service.Authorizer.CanUnlinkClassResources(actor)); err != nil {
		return err
	}

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
