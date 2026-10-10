package services

import (
	"fmt"
	"slices"

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
	if err := service.validateACAccess(service_model.CreateACParams{UserID: principal.UserID, ProblemID: problemID}, principal); err != nil {
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
	if err := service.validateACAccess(service_model.CreateACParams{UserID: principal.UserID, ProblemID: problemID}, principal); err != nil {
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

func (service *SolutionService) validateACAccess(params service_model.CreateACParams, student service_model.Principal) error {
	if params.UserID == "" || params.ProblemID <= 0 {
		return serviceerr.NewValidationError("user ID and positive problem ID are required", nil)
	}

	// Load the same validation data used by batches for this single target.
	target := data.ACKey{UserID: params.UserID, ProblemID: params.ProblemID}
	validationDataByAC, err := service.Datastore.GetACValidationData([]data.ACKey{target})
	if err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	_, err = prepareAC(params, student, validationDataByAC[target])
	return err
}

// prepareAC validates one request using resolved validation data, without datastore reads.
// Individual commands and batches share its target, identity and access rules.
func prepareAC(
	params service_model.CreateACParams,
	student service_model.Principal,
	validationData data.ACValidationData,
) (service_model.PreparedAC, error) {
	if validationData.Problem == nil {
		return service_model.PreparedAC{}, serviceerr.NewProblemNotFoundError(fmt.Sprintf("problem %d does not exist", params.ProblemID), nil)
	}
	if validationData.Class == nil {
		return service_model.PreparedAC{}, serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("class %d does not exist", validationData.Problem.ClassID), nil)
	}
	if student.UserID == "" {
		return service_model.PreparedAC{}, serviceerr.NewInternalError("could not verify student "+params.UserID, nil)
	}
	if student.UserID != params.UserID {
		return service_model.PreparedAC{}, serviceerr.NewInternalError("verified student identity mismatch", nil)
	}
	for _, roleID := range validationData.AccessRoleIDs {
		if slices.Contains(student.Roles, roleID) {
			return service_model.PreparedAC{
				User:            student.User,
				Problem:         mapDataProblemToServiceProblem(*validationData.Problem),
				Class:           mapDataClassToServiceClass(*validationData.Class),
				AlreadyRecorded: validationData.AlreadyRecorded,
			}, nil
		}
	}
	return service_model.PreparedAC{}, serviceerr.NewUserNotEnrolledError(fmt.Sprintf("user %s has no access to class %s", params.UserID, validationData.Class.ClassCode), nil)
}

// PrepareCreationBatch validates the complete batch with grouped datastore reads.
// Student identities and role snapshots are resolved by the caller. Existing ACs
// are marked for preview, while repeated entries remain in their submitted order.
func (service *SolutionService) PrepareCreationBatch(
	inputs []service_model.CreateACParams,
	studentsByID map[string]service_model.Principal,
) ([]service_model.PreparedAC, []service_model.BatchEntryError, error) {
	normalizedInputs := make([]service_model.CreateACParams, len(inputs))
	entryErrors := make([]error, len(inputs))
	var requestedACs []data.ACKey

	// Normalize values and collect the pairs eligible for datastore reads.
	for index, input := range inputs {
		normalized, err := service_model.NormalizeAndValidateCreateACParams(input)
		if err != nil {
			entryErrors[index] = serviceerr.NewValidationError("invalid AC values", err)
			continue
		}
		normalizedInputs[index] = normalized
		requestedACs = append(requestedACs, data.ACKey{UserID: normalized.UserID, ProblemID: normalized.ProblemID})
	}

	// Fetch the validation data; a technical failure aborts preparation before writes.
	var validationDataByAC map[data.ACKey]data.ACValidationData
	if len(requestedACs) > 0 {
		var err error
		validationDataByAC, err = service.Datastore.GetACValidationData(requestedACs)
		if err != nil {
			return nil, nil, serviceerr.MakeGeneralError(err)
		}
	}

	// Validate each AC in input order and report its original entry number.
	prepared := make([]service_model.PreparedAC, len(inputs))
	var issues []service_model.BatchEntryError
	for index, input := range normalizedInputs {
		if entryErrors[index] == nil {
			key := data.ACKey{UserID: input.UserID, ProblemID: input.ProblemID}
			prepared[index], entryErrors[index] = prepareAC(input, studentsByID[input.UserID], validationDataByAC[key])
		}
		if entryErrors[index] != nil {
			issues = append(issues, service_model.BatchEntryError{Entry: index + 1, Err: entryErrors[index]})
			continue
		}
		prepared[index].Entry = index + 1
	}
	if len(issues) > 0 {
		return nil, issues, nil
	}
	return prepared, nil, nil
}

// ExecuteCreationBatch revalidates all entries before one atomic datastore write.
// No concurrent changes are assumed between validation and execution.
func (service *SolutionService) ExecuteCreationBatch(batch []service_model.PreparedAC, studentsByID map[string]service_model.Principal) (service_model.ACBatchResult, []service_model.BatchEntryError, error) {
	var result service_model.ACBatchResult

	// Resolve current targets, permissions and recorded solves for the whole batch.
	inputs := make([]service_model.CreateACParams, len(batch))
	for index, entry := range batch {
		inputs[index] = service_model.CreateACParams{UserID: entry.User.UserID, ProblemID: entry.Problem.ID}
	}

	prepared, issues, err := service.PrepareCreationBatch(inputs, studentsByID)
	if err != nil {
		return result, nil, serviceerr.NewBatchPreparationError("could not revalidate AC batch", err)
	}

	if len(issues) > 0 {
		return result, issues, nil
	}

	// Reject changed destinations before applying any operation.
	for index, entry := range prepared {
		if entry.Class.ID != batch[index].Class.ID {
			issues = append(issues, service_model.BatchEntryError{Entry: index + 1, Err: serviceerr.NewConflictError("problem class changed; validate a new batch", nil)})
		}
	}

	if len(issues) > 0 {
		return result, issues, nil
	}

	// Persist the validated pairs together and count actual new registrations.
	targets := make([]data.ACKey, len(prepared))
	for index, entry := range prepared {
		targets[index] = data.ACKey{UserID: entry.User.UserID, ProblemID: entry.Problem.ID}
	}

	insertedCount, err := service.Datastore.AddACBatch(targets)
	if err != nil {
		return result, nil, serviceerr.MakeGeneralError(err)
	}

	result.Recorded = insertedCount
	result.AlreadyRecorded = len(prepared) - insertedCount
	return result, nil, nil
}
