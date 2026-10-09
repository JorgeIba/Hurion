package services

import (
	"errors"
	"fmt"

	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// ExecuteProblemBatch revalidates the complete snapshot before the first write.
// The caller must authorize the actor and claim the batch before calling it.
// Validation issues imply no writes. An execution error stops subsequent writes
// but does not roll back earlier ones; the result contains acknowledged creations.
// Execution assumes no concurrent changes between validation and writes.
func (service *BatchService) ExecuteProblemBatch(batch []service_model.PreparedProblem) (service_model.ProblemBatchResult, []service_model.BatchEntryError, error) {
	var result service_model.ProblemBatchResult
	if err := ValidateBatchSize(len(batch)); err != nil {
		return result, []service_model.BatchEntryError{{Err: err}}, nil
	}

	inputs := make([]service_model.CreateProblemParams, len(batch))
	for index, entry := range batch {
		inputs[index] = entry.Params
	}

	prepared, issues := service.PrepareProblemBatch(inputs)
	if len(issues) > 0 {
		return result, issues, nil
	}

	for index, entry := range prepared {
		if entry.Class.ID != batch[index].Class.ID {
			issues = append(issues, service_model.BatchEntryError{Entry: index + 1, Err: serviceerr.NewConflictError("destination class changed; validate a new batch", nil)})
		}
	}

	if len(issues) > 0 {
		return result, issues, nil
	}

	for index, entry := range prepared {
		problemID, err := service.Problem.CreateProblemIntoClass(entry.Params)
		if err != nil {
			return result, nil, fmt.Errorf("could not create problem at batch entry %d: %w", index+1, err)
		}
		result.CreatedProblemIDs = append(result.CreatedProblemIDs, problemID)
	}
	return result, nil, nil
}

// ExecuteACBatch revalidates every entry using current students resolved by the
// caller before writing. Existing and repeated ACs are reported without blocking.
// The caller must authorize the actor and claim the batch before calling it.
// Validation issues imply no writes. Execution errors stop subsequent writes
// without rollback; the failed operation may also have created a user record.
// Execution assumes no concurrent changes between validation and writes.
func (service *BatchService) ExecuteACBatch(batch []service_model.PreparedAC, studentsByID map[string]service_model.Principal) (service_model.ACBatchResult, []service_model.BatchEntryError, error) {
	var result service_model.ACBatchResult
	if err := ValidateBatchSize(len(batch)); err != nil {
		return result, []service_model.BatchEntryError{{Err: err}}, nil
	}

	inputs := make([]service_model.CreateACParams, len(batch))
	for index, entry := range batch {
		inputs[index] = service_model.CreateACParams{UserID: entry.User.UserID, ProblemID: entry.Problem.ID}
	}

	prepared, issues := service.PrepareACBatch(inputs, studentsByID)
	if len(issues) > 0 {
		return result, issues, nil
	}

	for index, entry := range prepared {
		if entry.Class.ID != batch[index].Class.ID {
			issues = append(issues, service_model.BatchEntryError{Entry: index + 1, Err: serviceerr.NewConflictError("problem class changed; validate a new batch", nil)})
		}
	}

	if len(issues) > 0 {
		return result, issues, nil
	}

	for index, entry := range prepared {
		if entry.AlreadyRecorded {
			result.AlreadyRecorded++
			continue
		}
		student := studentsByID[entry.User.UserID]
		err := service.Solution.AddACByPrincipal(student, entry.Problem.ID)
		if _, alreadyRecorded := errors.AsType[*serviceerr.ACAlreadyExistsError](err); alreadyRecorded {
			result.AlreadyRecorded++
			continue
		}
		if err != nil {
			return result, nil, fmt.Errorf("could not record AC at batch entry %d: %w", index+1, err)
		}
		result.Recorded++
	}
	return result, nil, nil
}
