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

	// 1. Business Logic
	if hasUserSolvedAlready, err := service.Datastore.HasUserSolvedProblem(userID, problemID); err != nil {
		return serviceerr.MakeGeneralError(fmt.Errorf("error checking if user %s has solved problem %d: %w", userID, problemID, err))
	} else if hasUserSolvedAlready {
		return serviceerr.NewConflictError(fmt.Sprintf("user %s has already solved problem %d", userID, problemID), nil)
	}

	if err := service.Datastore.AddAC(userID, problemID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}

	return nil
}

// AddACByPrincipal adds an AC for a user (Authenticated enrollment version).
func (service *SolutionService) AddACByPrincipal(principal service_model.Principal, problemID int64) error {
	allowed, err := service.authorizeAC(principal, problemID)
	if err != nil {
		return err
	}
	if !allowed {
		return serviceerr.NewUserNotEnrolledError(fmt.Sprintf("user %s cannot AC the problem %d (missing required roles)", principal.UserID, problemID), nil)
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
	allowed, err := service.authorizeAC(principal, problemID)
	if err != nil {
		return err
	}
	if !allowed {
		return serviceerr.NewUserNotEnrolledError(fmt.Sprintf("user %s cannot propose AC for problem %d (missing required roles)", principal.UserID, problemID), nil)
	}

	// Layer 2 Safeguard
	if err := service.Datastore.EnsureUser(data.User{UserID: principal.UserID, DisplayName: principal.DisplayName}); err != nil {
		return serviceerr.MakeGeneralError(fmt.Errorf("failed to ensure user %s exists: %w", principal.UserID, err))
	}

	// 2. Business Logic
	if hasUserSolvedAlready, err := service.Datastore.HasUserSolvedProblem(principal.UserID, problemID); err != nil {
		return serviceerr.MakeGeneralError(fmt.Errorf("error checking if user %s has solved problem %d: %w", principal.UserID, problemID, err))
	} else if hasUserSolvedAlready {
		return serviceerr.NewConflictError(fmt.Sprintf("user %s has already solved problem %d", principal.UserID, problemID), nil)
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

// authorizeAC centralizes the logic to verify if a principal is allowed to AC a specific problem.
func (service *SolutionService) authorizeAC(principal service_model.Principal, problemID int64) (bool, error) {
	// 1. Resolve Context
	problem, err := service.Datastore.GetProblemByID(problemID)
	if err != nil {
		if dataerr.IsNotFound(err) {
			return false, serviceerr.NewProblemNotFoundError(fmt.Sprintf("problem %d doesn't exist", problemID), err)
		}
		return false, serviceerr.MakeGeneralError(err)
	}

	class, err := service.Datastore.GetClassByClassID(problem.ClassID)
	if err != nil {
		return false, serviceerr.MakeGeneralError(err)
	}

	accessRoles, err := service.Datastore.GetClassAccessRolesByClassCode(class.ClassCode)
	if err != nil {
		return false, serviceerr.MakeGeneralError(err)
	}

	// 2. Authorize
	isAuthorized := false
	roleSet := make(map[string]bool, len(principal.Roles))
	for _, r := range principal.Roles {
		roleSet[r] = true
	}
	for _, ar := range accessRoles {
		if roleSet[ar.RoleID] {
			isAuthorized = true
			break
		}
	}

	return isAuthorized, nil
}
