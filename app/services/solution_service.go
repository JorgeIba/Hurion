package services

import (
	"fmt"

	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// SolutionService handles AC (accepted solution) operations
type SolutionService struct {
	Datastore data.DataStore
}

// AddACToUserID adds an AC (accepted solution) for a user.
// This is a trusted method (no enrollment check).
func (service *SolutionService) AddACToUserID(userID string, problemID int64) error {
	// Layer 2 Safeguard: Ensure user exists before linking foreign keys
	if err := service.Datastore.EnsureUser(data.User{UserID: userID, DisplayName: "Unknown"}); err != nil {
		return serviceerr.MakeGeneralError(fmt.Errorf("failed to ensure user %s exists: %w", userID, err))
	}

	if err := service.validateACNotSolved(userID, problemID); err != nil {
		return err
	}

	if err := service.Datastore.AddAC(userID, problemID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}

	return nil
}

// AddACByPrincipal adds an AC for a user (Authenticated enrollment version).
func (service *SolutionService) AddACByPrincipal(principal service_model.Principal, problemID int64) error {
	if _, err := service.validateACAccess(service_model.CreateACParams{UserID: principal.UserID, ProblemID: problemID}, principal); err != nil {
		return err
	}
	return service.AddACToUserID(principal.UserID, problemID)
}

// UndoAC removes an AC for a user.
func (service *SolutionService) UndoAC(userID string, problemID int64) error {
	if hasUserSolvedAlready, err := service.Datastore.HasUserSolvedProblem(userID, problemID); err != nil {
		return serviceerr.MakeGeneralError(fmt.Errorf("error checking if user %s has solved problem %d: %w", userID, problemID, err))
	} else if !hasUserSolvedAlready {
		return serviceerr.NewConflictError(fmt.Sprintf("user %s has not solved problem %d", userID, problemID), nil)
	}

	if err := service.Datastore.RemoveAC(userID, problemID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

// AddPendingACByPrincipal adds a pending AC confirmation
func (service *SolutionService) AddPendingACByPrincipal(messageID string, principal service_model.Principal, problemID int64) error {
	if _, err := service.validateACAccess(service_model.CreateACParams{UserID: principal.UserID, ProblemID: problemID}, principal); err != nil {
		return err
	}
	if err := service.validateACNotSolved(principal.UserID, problemID); err != nil {
		return err
	}

	// Layer 2 Safeguard
	if err := service.Datastore.EnsureUser(data.User{UserID: principal.UserID, DisplayName: principal.DisplayName}); err != nil {
		return serviceerr.MakeGeneralError(fmt.Errorf("failed to ensure user %s exists: %w", principal.UserID, err))
	}

	if err := service.Datastore.AddPendingConfirmation(mapServicePendingConfirmationToDataPendingConfirmation(service_model.PendingConfirmation{
		MessageID: messageID,
		UserID:    principal.UserID,
		ProblemID: problemID,
	})); err != nil {
		return serviceerr.MakeGeneralError(err)
	}

	return nil
}

// ExistsPendingConfirmation checks if a pending confirmation exists for a message ID
func (service *SolutionService) ExistsPendingConfirmation(messageID string) (bool, error) {
	exists, err := service.Datastore.ExistsPendingConfirmation(messageID)
	if err != nil {
		return false, serviceerr.MakeGeneralError(err)
	}
	return exists, nil
}

// GetPendingConfirmationByMessageID retrieves a pending confirmation by message ID
func (service *SolutionService) GetPendingConfirmationByMessageID(messageID string) (*service_model.PendingConfirmation, error) {
	userID, problemID, err := service.Datastore.GetPendingConfirmation(messageID)

	if err != nil {
		switch {
		case dataerr.IsNotFound(err):
			return &service_model.PendingConfirmation{}, serviceerr.NewNotFoundInfoError("", err)
		default:
			return &service_model.PendingConfirmation{}, serviceerr.MakeGeneralError(err)
		}
	}

	return &service_model.PendingConfirmation{MessageID: messageID, UserID: userID, ProblemID: problemID}, nil
}

// AcceptACProposal accepts a pending AC proposal
func (service *SolutionService) AcceptACProposal(messageID string) error {
	pendingConfirmation, err := service.GetPendingConfirmationByMessageID(messageID)
	if err != nil {
		return err
	}

	if err := service.Datastore.RemovePendingConfirmation(messageID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}

	// We trust the DB record.
	return service.AddACToUserID(pendingConfirmation.UserID, pendingConfirmation.ProblemID)
}

// RejectACProposal rejects a pending AC proposal
func (service *SolutionService) RejectACProposal(messageID string) error {
	if err := service.Datastore.RemovePendingConfirmation(messageID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

// ValidateAC validates a possible AC without creating users or recording solves.
// params identifies the requested target; student supplies the resolved identity
// and roles. The returned snapshot must be revalidated before deferred execution.
func (service *SolutionService) ValidateAC(params service_model.CreateACParams, student service_model.Principal) (service_model.PreparedAC, error) {
	params, err := service_model.NormalizeAndValidateCreateACParams(params)
	if err != nil {
		return service_model.PreparedAC{}, serviceerr.NewValidationError("invalid AC values", err)
	}
	prepared, err := service.validateACAccess(params, student)
	if err != nil {
		return service_model.PreparedAC{}, err
	}
	if err := service.validateACNotSolved(params.UserID, params.ProblemID); err != nil {
		return prepared, err
	}
	return prepared, nil
}

func (service *SolutionService) validateACNotSolved(userID string, problemID int64) error {
	solved, err := service.Datastore.HasUserSolvedProblem(userID, problemID)
	if err != nil {
		return serviceerr.MakeGeneralError(fmt.Errorf("error checking if user %s has solved problem %d: %w", userID, problemID, err))
	}
	if solved {
		return serviceerr.NewACAlreadyExistsError(fmt.Sprintf("user %s has already solved problem %d", userID, problemID), nil)
	}
	return nil
}

func (service *SolutionService) validateACAccess(params service_model.CreateACParams, student service_model.Principal) (service_model.PreparedAC, error) {
	var result service_model.PreparedAC
	if params.UserID == "" || params.ProblemID <= 0 {
		return result, serviceerr.NewValidationError("user ID and positive problem ID are required", nil)
	}
	problem, err := service.Datastore.GetProblemByID(params.ProblemID)
	if err != nil {
		if dataerr.IsNotFound(err) {
			return result, serviceerr.NewProblemNotFoundError(fmt.Sprintf("problem %d does not exist", params.ProblemID), err)
		}
		return result, serviceerr.MakeGeneralError(err)
	}
	class, err := service.Datastore.GetClassByClassID(problem.ClassID)
	if err != nil {
		if dataerr.IsNotFound(err) {
			return result, serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("class %d does not exist", problem.ClassID), err)
		}
		return result, serviceerr.MakeGeneralError(err)
	}
	if student.UserID == "" {
		return result, serviceerr.NewInternalError("could not verify student "+params.UserID, nil)
	}
	if student.UserID != params.UserID {
		return result, serviceerr.NewInternalError("verified student identity mismatch", nil)
	}
	resolvedClass := mapDataClassToServiceClass(class)
	accessRoles, err := service.Datastore.GetClassAccessRolesByClassCode(class.ClassCode)
	if err != nil {
		return result, serviceerr.MakeGeneralError(err)
	}
	roleSet := make(map[string]bool, len(student.Roles))
	for _, roleID := range student.Roles {
		roleSet[roleID] = true
	}
	allowed := false
	for _, accessRole := range accessRoles {
		if roleSet[accessRole.RoleID] {
			allowed = true
			break
		}
	}
	if !allowed {
		return result, serviceerr.NewUserNotEnrolledError(fmt.Sprintf("user %s has no access to class %s", params.UserID, resolvedClass.ClassCode), nil)
	}
	return service_model.PreparedAC{User: student.User, Problem: mapDataProblemToServiceProblem(problem), Class: resolvedClass}, nil
}
