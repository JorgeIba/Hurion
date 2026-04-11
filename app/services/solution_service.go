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
	dataNotFoundErrSolution = &dataerr.NotFoundError{}
)

// SolutionService handles AC (accepted solution) operations
type SolutionService struct {
	Datastore data.DataStore
}

// AddACToUser adds an AC (accepted solution) for a user
func (service *SolutionService) AddACToUser(userID string, problemID int64) error {
	// Validate problem exists
	if isProblemPresent, err := helpers.ExistsProblem(service.Datastore, problemID); err != nil {
		return err
	} else if !isProblemPresent {
		return serviceerr.NewProblemNotFoundError(fmt.Sprintf("are you sure problem %d exists?", problemID), nil)
	}

	// Validate user is registered in the class of the problem
	if isBroAbleToAC, err := helpers.CanBroACTheProblem(service.Datastore, userID, problemID); err != nil {
		return err
	} else if !isBroAbleToAC {
		return serviceerr.NewUserNotEnrolledError(fmt.Sprintf("bro %s cannot AC the problem with id: %d", userID, problemID), nil)
	}

	// ACing
	if err := service.Datastore.AddAC(userID, problemID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}

	return nil
}

// UndoACByProblemAndUserID removes an AC for a user
func (service *SolutionService) UndoACByProblemAndUserID(problemID int64, userID string) error {
	// Validate problem exists
	if isProblemPresent, err := helpers.ExistsProblem(service.Datastore, problemID); err != nil {
		return err
	} else if !isProblemPresent {
		return serviceerr.NewProblemNotFoundError(fmt.Sprintf("problem %d doesn't exist", problemID), nil)
	}

	// Validate user is registered in the class of the problem
	if isBroAbleToUndoAC, err := helpers.HasUserSolvedProblem(service.Datastore, userID, problemID); err != nil {
		return err
	} else if !isBroAbleToUndoAC {
		return serviceerr.NewConflictError(fmt.Sprintf("user %s has not solved problem %d or is not enrolled in the class", userID, problemID), nil)
	}

	// Undo-ing
	if err := service.Datastore.RemoveAC(userID, problemID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

// AddACToUsername adds an AC for a user by username
func (service *SolutionService) AddACToUsername(userName string, problemID int64) error {
	user, err := service.Datastore.GetUserByUserName(userName)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErrSolution):
			return serviceerr.NewUserNotFoundError(fmt.Sprintf("username %s not found", userName), err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	return service.AddACToUser(user.UserID, problemID)
}

// UndoACByProblemAndUsername removes an AC for a user by username
func (service *SolutionService) UndoACByProblemAndUsername(problemID int64, userName string) error {
	user, err := service.Datastore.GetUserByUserName(userName)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErrSolution):
			return serviceerr.NewUserNotFoundError(fmt.Sprintf("username %s not found", userName), err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	err = service.UndoACByProblemAndUserID(problemID, user.UserID)
	return err
}

// AddPendingACOfUserID adds a pending AC confirmation
func (service *SolutionService) AddPendingACOfUserID(messageID, userID string, problemID int64) error {
	// Validate problem exists
	if isProblemPresent, err := helpers.ExistsProblem(service.Datastore, problemID); err != nil {
		return err
	} else if !isProblemPresent {
		return serviceerr.NewProblemNotFoundError(fmt.Sprintf("are you sure problem %d exists?", problemID), nil)
	}

	// Validate user is registered in the class of the problem
	if isBroAbleToAC, err := helpers.CanBroACTheProblem(service.Datastore, userID, problemID); err != nil {
		return err
	} else if !isBroAbleToAC {
		return serviceerr.NewUserNotEnrolledError(fmt.Sprintf("bro %s cannot AC the problem with id: %d", userID, problemID), nil)
	}

	if err := service.Datastore.AddPendingConfirmation(mapServicePendingConfirmationToDataPendingConfirmation(service_model.PendingConfirmation{
		MessageID: messageID,
		UserID:    userID,
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
		case errors.As(err, &dataNotFoundErrSolution):
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

	if err := service.AddACToUser(pendingConfirmation.UserID, pendingConfirmation.ProblemID); err != nil {
		return err
	}

	return nil
}

// RejectACProposal rejects a pending AC proposal
func (service *SolutionService) RejectACProposal(messageID string) error {
	if err := service.Datastore.RemovePendingConfirmation(messageID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}
