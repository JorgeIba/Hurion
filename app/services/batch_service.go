package services

import (
	"errors"
	"fmt"

	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// BatchService prepares previews using queries only. Actor authorization belongs
// to the caller; preparation neither creates users nor changes their solves.
type BatchService struct {
	Problem  ProblemService
	Solution *SolutionService
}

// ValidateBatchSize checks the shared operation limit before preparing a batch.
func ValidateBatchSize(size int) error {
	if size < 1 || size > service_model.MaxBatchOperations {
		return serviceerr.NewValidationError(fmt.Sprintf("batch must contain between 1 and %d operations", service_model.MaxBatchOperations), nil)
	}
	return nil
}

// PrepareProblemBatch returns the complete prepared batch or indexed errors,
// never partial results. Entry numbers are one-based; zero denotes a batch-wide error.
func (service *BatchService) PrepareProblemBatch(inputs []service_model.CreateProblemParams) ([]service_model.PreparedProblem, []service_model.BatchEntryError) {
	if err := ValidateBatchSize(len(inputs)); err != nil {
		return nil, []service_model.BatchEntryError{{Err: err}}
	}
	var prepared []service_model.PreparedProblem
	var issues []service_model.BatchEntryError
	for index, input := range inputs {
		preparedProblem, err := service.Problem.PrepareCreation(input)
		if err != nil {
			issues = append(issues, service_model.BatchEntryError{Entry: index + 1, Err: err})
			continue
		}
		preparedProblem.Entry = index + 1
		prepared = append(prepared, preparedProblem)
	}
	if len(issues) > 0 {
		return nil, issues
	}
	return prepared, nil
}

// PrepareACBatch returns the complete prepared batch or indexed errors, never
// partial results. Entry numbers are one-based; zero denotes a batch-wide error.
// studentsByID contains the identities and role snapshots supplied by the controller.
// Already recorded ACs are marked for preview; they do not invalidate the batch.
func (service *BatchService) PrepareACBatch(inputs []service_model.CreateACParams, studentsByID map[string]service_model.Principal) ([]service_model.PreparedAC, []service_model.BatchEntryError) {
	if err := ValidateBatchSize(len(inputs)); err != nil {
		return nil, []service_model.BatchEntryError{{Err: err}}
	}
	var prepared []service_model.PreparedAC
	var issues []service_model.BatchEntryError
	for index, input := range inputs {
		student := studentsByID[service_model.NormalizeACUserID(input.UserID)]
		preparedAC, err := service.Solution.ValidateAC(input, student)
		if err != nil {
			if _, alreadyRecorded := errors.AsType[*serviceerr.ACAlreadyExistsError](err); !alreadyRecorded {
				issues = append(issues, service_model.BatchEntryError{Entry: index + 1, Err: err})
				continue
			}
			preparedAC.AlreadyRecorded = true
		}
		preparedAC.Entry = index + 1
		prepared = append(prepared, preparedAC)
	}
	if len(issues) > 0 {
		return nil, issues
	}
	return prepared, nil
}
