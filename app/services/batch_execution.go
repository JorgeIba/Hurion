package services

import (
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// ExecuteProblemBatch revalidates the complete snapshot before the first write.
// The caller must authorize the actor and claim the batch before calling it.
// Validation issues imply no writes. An execution error stops subsequent writes
// but does not roll back earlier ones; the result contains acknowledged creations.
// Execution assumes no concurrent changes between validation and writes.
func (service *BatchService) ExecuteProblemBatch(batch []service_model.PreparedProblem) (service_model.ProblemCreateBatchResult, []service_model.BatchEntryError, error) {
	if err := ValidateBatchSize(len(batch)); err != nil {
		return service_model.ProblemCreateBatchResult{}, []service_model.BatchEntryError{{Err: err}}, nil
	}
	return service.Problem.ExecuteCreationBatch(batch)
}

// ExecuteACBatch revalidates every entry using current students resolved by the
// caller before writing. Existing and repeated ACs are reported without blocking.
// The caller must authorize the actor and claim the batch before calling it.
// Validation issues imply no writes. User creation and AC registration are atomic;
// the result counts registrations acknowledged after commit.
// Execution assumes no concurrent changes between validation and writes.
func (service *BatchService) ExecuteACBatch(batch []service_model.PreparedAC, studentsByID map[string]service_model.Principal) (service_model.ACBatchResult, []service_model.BatchEntryError, error) {
	var result service_model.ACBatchResult
	if err := ValidateBatchSize(len(batch)); err != nil {
		return result, []service_model.BatchEntryError{{Err: err}}, nil
	}

	return service.Solution.ExecuteCreationBatch(batch, studentsByID)
}

// ExecuteProblemUpdateBatch revalidates the complete snapshot before the first write.
// The caller must authorize the actor and claim the batch before calling it.
// Validation issues imply no writes. An execution error stops subsequent writes
// but does not roll back earlier ones; the result contains acknowledged updates.
// Execution assumes no concurrent changes between validation and writes.
func (service *BatchService) ExecuteProblemUpdateBatch(batch []service_model.PreparedProblemUpdate) (service_model.ProblemUpdateBatchResult, []service_model.BatchEntryError, error) {
	if err := ValidateBatchSize(len(batch)); err != nil {
		return service_model.ProblemUpdateBatchResult{}, []service_model.BatchEntryError{{Err: err}}, nil
	}
	return service.Problem.ExecuteUpdateBatch(batch)
}
