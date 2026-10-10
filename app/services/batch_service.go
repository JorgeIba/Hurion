package services

import (
	"fmt"

	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// BatchService prepares and executes batches through the problem and solution
// services. Actor authorization belongs to the caller; preparation is read-only.
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
	return service.Problem.PrepareCreationBatch(inputs)
}

// PrepareACBatch returns the complete prepared batch or indexed errors, never
// partial results. Entry numbers are one-based; zero denotes a batch-wide error.
// studentsByID contains the identities and role snapshots supplied by the controller.
// Already recorded ACs are marked for preview; they do not invalidate the batch.
func (service *BatchService) PrepareACBatch(inputs []service_model.CreateACParams, studentsByID map[string]service_model.Principal) ([]service_model.PreparedAC, []service_model.BatchEntryError, error) {
	if err := ValidateBatchSize(len(inputs)); err != nil {
		return nil, []service_model.BatchEntryError{{Err: err}}, nil
	}
	return service.Solution.PrepareCreationBatch(inputs, studentsByID)
}

// PrepareProblemUpdateBatch returns the complete prepared batch or indexed errors,
// never partial results. Entry numbers are one-based; zero denotes a batch-wide error.
func (service *BatchService) PrepareProblemUpdateBatch(inputs []service_model.UpdateProblemParams) ([]service_model.PreparedProblemUpdate, []service_model.BatchEntryError) {
	if err := ValidateBatchSize(len(inputs)); err != nil {
		return nil, []service_model.BatchEntryError{{Err: err}}
	}
	return service.Problem.PrepareUpdateBatch(inputs)
}
