package services

import (
	"fmt"
	"slices"

	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/services/helpers"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// SolutionService handles AC (accepted solution) operations
type SolutionService struct {
	Datastore  data.DataStore
	Authorizer Authorizer
}

// GrantACWithoutEnrollmentCheck records an AC without checking student enrollment.
// The actor must still be authorized for the problem class.
func (service *SolutionService) GrantACWithoutEnrollmentCheck(actor service_model.Actor, userID string, problemID int64) error {
	if err := service.requireProblemManagement(actor, problemID); err != nil {
		return err
	}

	if err := service.validateACNotSolved(userID, problemID); err != nil {
		return err
	}
	return service.recordAC(userID, problemID)
}

func (service *SolutionService) recordAC(userID string, problemID int64) error {
	// Layer 2 Safeguard: Ensure user exists before linking foreign keys
	if err := service.Datastore.EnsureUser(data.User{UserID: userID, DisplayName: "Unknown"}); err != nil {
		return serviceerr.MakeGeneralError(fmt.Errorf("failed to ensure user %s exists: %w", userID, err))
	}

	if err := service.Datastore.AddAC(userID, problemID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}

	return nil
}

// GrantAC authorizes the actor and validates the student's class access before recording an AC.
func (service *SolutionService) GrantAC(actor service_model.Actor, student service_model.UserWithRoles, problemID int64) error {
	prepared, err := service.validateACAccess(service_model.CreateACParams{UserID: student.UserID, ProblemID: problemID}, student)
	if err != nil {
		return err
	}

	if err := requireManageClasses(service.Authorizer, actor, prepared.Class.ClassCode); err != nil {
		return err
	}

	if prepared.AlreadyRecorded {
		return serviceerr.NewACAlreadyExistsError(fmt.Sprintf("user %s has already solved problem %d", student.UserID, problemID), nil)
	}

	return service.recordAC(student.UserID, problemID)
}

// UndoAC removes an AC for a user.
func (service *SolutionService) UndoAC(actor service_model.Actor, userID string, problemID int64) error {
	if actor.Kind != service_model.ActorUser || actor.ID == "" || actor.ID != userID {
		if err := service.requireProblemManagement(actor, problemID); err != nil {
			return err
		}
	}

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

// ProposeAC validates and creates or replaces a proposal. referenceID
// is opaque to the service: either a prefix message ID or a slash interaction ID.
func (service *SolutionService) ProposeAC(referenceID string, student service_model.UserWithRoles, problemID int64) (service_model.Problem, error) {
	prepared, err := service.validateACAccess(service_model.CreateACParams{UserID: student.UserID, ProblemID: problemID}, student)
	if err != nil {
		return service_model.Problem{}, err
	}

	if prepared.AlreadyRecorded {
		return service_model.Problem{}, serviceerr.NewACAlreadyExistsError(fmt.Sprintf("user %s has already solved problem %d", student.UserID, problemID), nil)
	}

	if err := service.Datastore.EnsureUser(data.User{UserID: student.UserID, DisplayName: student.DisplayName}); err != nil {
		return service_model.Problem{}, serviceerr.MakeGeneralError(err)
	}

	written, err := service.Datastore.AddPendingConfirmation(mapServicePendingConfirmationToDataPendingConfirmation(
		service_model.PendingConfirmation{
			MessageID: referenceID,
			UserID:    student.UserID,
			ProblemID: problemID,
		}),
	)

	if err != nil {
		return service_model.Problem{}, serviceerr.MakeGeneralError(err)
	}

	if !written {
		return service_model.Problem{}, serviceerr.NewACAlreadyExistsError(fmt.Sprintf("user %s has already solved problem %d", student.UserID, problemID), nil)
	}

	return prepared.Problem, nil
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

// AcceptACProposal returns the proposal only after its approval is committed.
func (service *SolutionService) AcceptACProposal(actor service_model.Actor, referenceID string) (service_model.PendingConfirmation, error) {
	proposal, err := service.GetPendingConfirmationByMessageID(referenceID)
	if err != nil {
		return service_model.PendingConfirmation{}, err
	}

	if err := service.requireProblemManagement(actor, proposal.ProblemID); err != nil {
		return service_model.PendingConfirmation{}, err
	}
	pending, err := service.Datastore.AcceptPendingConfirmation(referenceID)
	if err != nil {
		if dataerr.IsNotFound(err) {
			return service_model.PendingConfirmation{}, serviceerr.NewNotFoundInfoError("proposal was resolved or replaced", err)
		}

		return service_model.PendingConfirmation{}, serviceerr.MakeGeneralError(err)
	}

	return mapDataPendingConfirmationToServicePendingConfirmation(pending), nil
}

// RejectACProposal returns the proposal that was actually removed.
func (service *SolutionService) RejectACProposal(actor service_model.Actor, referenceID string) (service_model.PendingConfirmation, error) {
	proposal, err := service.GetPendingConfirmationByMessageID(referenceID)
	if err != nil {
		return service_model.PendingConfirmation{}, err
	}

	if err := service.requireProblemManagement(actor, proposal.ProblemID); err != nil {
		return service_model.PendingConfirmation{}, err
	}
	pending, err := service.Datastore.RemovePendingConfirmation(referenceID)
	if err != nil {
		if dataerr.IsNotFound(err) {
			return service_model.PendingConfirmation{}, serviceerr.NewNotFoundInfoError("proposal was resolved or replaced", err)
		}

		return service_model.PendingConfirmation{}, serviceerr.MakeGeneralError(err)
	}

	return mapDataPendingConfirmationToServicePendingConfirmation(pending), nil
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

func (service *SolutionService) validateACAccess(params service_model.CreateACParams, student service_model.UserWithRoles) (service_model.PreparedAC, error) {
	if params.UserID == "" || params.ProblemID <= 0 {
		return service_model.PreparedAC{}, serviceerr.NewValidationError("user ID and positive problem ID are required", nil)
	}

	// Load the same validation data used by batches for this single target.
	target := data.ACKey{UserID: params.UserID, ProblemID: params.ProblemID}
	validationDataByAC, err := service.Datastore.GetACValidationData([]data.ACKey{target})
	if err != nil {
		return service_model.PreparedAC{}, serviceerr.MakeGeneralError(err)
	}

	return prepareAC(params, student, validationDataByAC[target])
}

// prepareAC validates one request using resolved validation data, without datastore reads.
// Individual commands and batches share its target, identity and access rules.
func prepareAC(
	params service_model.CreateACParams,
	student service_model.UserWithRoles,
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
	actor service_model.Actor,
	inputs []service_model.CreateACParams,
	studentsByID map[string]service_model.UserWithRoles,
) ([]service_model.PreparedAC, []service_model.BatchEntryError, error) {
	if err := helpers.ValidateBatchSize(len(inputs)); err != nil {
		return nil, []service_model.BatchEntryError{{Err: err}}, nil
	}

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
	classes := make([]string, len(prepared))
	for index, entry := range prepared {
		classes[index] = entry.Class.ClassCode
	}

	if err := requireManageClasses(service.Authorizer, actor, classes...); err != nil {
		return nil, nil, err
	}

	return prepared, nil, nil
}

// ExecuteCreationBatch revalidates all entries before one atomic datastore write.
// No concurrent changes are assumed between validation and execution.
func (service *SolutionService) ExecuteCreationBatch(actor service_model.Actor, batch []service_model.PreparedAC, studentsByID map[string]service_model.UserWithRoles) (service_model.ACBatchResult, []service_model.BatchEntryError, error) {
	var result service_model.ACBatchResult
	if err := helpers.ValidateBatchSize(len(batch)); err != nil {
		return result, []service_model.BatchEntryError{{Err: err}}, nil
	}

	// Resolve current targets, permissions and recorded solves for the whole batch.
	inputs := make([]service_model.CreateACParams, len(batch))
	for index, entry := range batch {
		inputs[index] = service_model.CreateACParams{UserID: entry.User.UserID, ProblemID: entry.Problem.ID}
	}

	prepared, issues, err := service.PrepareCreationBatch(actor, inputs, studentsByID)
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
	users := make([]data.User, len(prepared))
	for index, entry := range prepared {
		targets[index] = data.ACKey{UserID: entry.User.UserID, ProblemID: entry.Problem.ID}
		users[index] = data.User{UserID: entry.User.UserID, DisplayName: entry.User.DisplayName}
	}

	insertedCount, err := service.Datastore.AddACBatch(targets, users)
	if err != nil {
		return result, nil, serviceerr.MakeGeneralError(err)
	}

	result.Recorded = insertedCount
	result.AlreadyRecorded = len(prepared) - insertedCount
	return result, nil, nil
}

func (service *SolutionService) requireProblemManagement(actor service_model.Actor, problemID int64) error {
	problem, err := service.Datastore.GetProblemByID(problemID)
	if err != nil {
		if dataerr.IsNotFound(err) {
			return serviceerr.NewProblemNotFoundError("could not resolve problem", err)
		}

		return serviceerr.MakeGeneralError(err)
	}

	class, err := service.Datastore.GetClassByClassID(problem.ClassID)
	if err != nil {
		if dataerr.IsNotFound(err) {
			return serviceerr.NewClassCodeNotFoundError("could not resolve problem class", err)
		}

		return serviceerr.MakeGeneralError(err)
	}

	return requireManageClasses(service.Authorizer, actor, class.ClassCode)
}
