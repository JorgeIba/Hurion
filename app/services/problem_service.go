package services

import (
	"fmt"
	"sort"

	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/services/helpers"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// ProblemService defines problem management operations
//
//go:generate go tool mockgen -destination=mock_services/mock_problem_service.go -package=mock_services . ProblemService
type ProblemService interface {
	PrepareCreation(params service_model.CreateProblemParams) (service_model.PreparedProblem, error)
	PrepareCreationBatch(params []service_model.CreateProblemParams) ([]service_model.PreparedProblem, []service_model.BatchEntryError, error)
	ExecuteCreationBatch(batch []service_model.PreparedProblem) (service_model.ProblemCreateBatchResult, []service_model.BatchEntryError, error)
	PrepareProblemReplacement(params service_model.ReplaceProblemParams) (service_model.PreparedProblemReplacement, error)
	PrepareProblemPatch(params service_model.PatchProblemParams) (service_model.PreparedProblemPatch, error)
	PrepareProblemPatchBatch(params []service_model.PatchProblemParams) ([]service_model.PreparedProblemPatch, []service_model.BatchEntryError)
	ExecuteProblemPatchBatch(batch []service_model.PreparedProblemPatch) (service_model.ProblemPatchBatchResult, []service_model.BatchEntryError, error)
	CreateProblemIntoClass(params service_model.CreateProblemParams) (int64, error)
	RemoveProblemFromClass(problemID int64) error
	ReplaceProblemByProblemID(params service_model.ReplaceProblemParams) error
	PatchProblemByProblemID(params service_model.PatchProblemParams) error
	GetProblemsGroupedByTopic(classCode string) ([]service_model.ProblemsByTopic, error)
	GetProblemsByClass(classCode string) ([]service_model.Problem, error)
	GetProblemByID(problemID int64) (service_model.Problem, error)
}

// problemService handles problem management operations
type problemService struct {
	Datastore data.DataStore
}

func NewProblemService(datastore data.DataStore) ProblemService {
	return &problemService{Datastore: datastore}
}

// GetProblemByID retrieves a problem by its unique ID
func (service *problemService) GetProblemByID(problemID int64) (service_model.Problem, error) {
	problem, err := service.Datastore.GetProblemByID(problemID)
	if err != nil {
		switch {
		case dataerr.IsNotFound(err):
			return service_model.Problem{}, serviceerr.NewProblemNotFoundError(fmt.Sprintf("problem %d doesn't exist", problemID), err)
		default:
			return service_model.Problem{}, serviceerr.MakeGeneralError(err)
		}
	}

	return mapDataProblemToServiceProblem(problem), nil
}

// CreateProblemIntoClass creates a new problem in a class
func (service *problemService) CreateProblemIntoClass(params service_model.CreateProblemParams) (int64, error) {
	prepared, err := service.PrepareCreation(params)
	if err != nil {
		return 0, err
	}
	problemID, err := service.createPreparedProblem(prepared)
	return problemID, err
}

func (service *problemService) createPreparedProblem(prepared service_model.PreparedProblem) (int64, error) {
	problemID, err := service.Datastore.AddProblem(mapCreateProblemParamsToDataProblem(prepared.Params, prepared.Class.ID))
	if err != nil {
		switch {
		case dataerr.IsRowConflict(err):
			return problemID, serviceerr.NewDuplicateProblemError("problem with the same name in the same class", err)
		default:
			return 0, serviceerr.MakeGeneralError(err)
		}
	}
	return problemID, nil
}

// PrepareCreation normalizes values and resolves the destination without writing.
// The snapshot must be revalidated before deferred execution.
func (service *problemService) PrepareCreation(params service_model.CreateProblemParams) (service_model.PreparedProblem, error) {
	var prepared service_model.PreparedProblem
	params, err := service_model.NormalizeAndValidateCreateProblemParams(params)
	if err != nil {
		return prepared, serviceerr.NewValidationError("invalid problem values", err)
	}

	class, err := service.Datastore.GetClassByClassCode(params.ClassCode)
	if err != nil {
		switch {
		case dataerr.IsNotFound(err):
			return prepared, serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("classCode %s doesn't match with any classID", params.ClassCode), err)
		default:
			return prepared, serviceerr.MakeGeneralError(err)
		}
	}

	return service_model.PreparedProblem{Params: params, Class: mapDataClassToServiceClass(class)}, nil
}

// PrepareCreationBatch validates all values and resolves their distinct classes in one read.
func (service *problemService) PrepareCreationBatch(inputs []service_model.CreateProblemParams) ([]service_model.PreparedProblem, []service_model.BatchEntryError, error) {
	if err := helpers.ValidateBatchSize(len(inputs)); err != nil {
		return nil, []service_model.BatchEntryError{{Err: err}}, nil
	}

	normalized := make([]service_model.CreateProblemParams, len(inputs))
	var issues []service_model.BatchEntryError
	classCodes := make([]string, 0, len(inputs))
	seenCodes := make(map[string]struct{}, len(inputs))
	// Normalize every entry and collect each destination class once.
	for index, input := range inputs {
		value, err := service_model.NormalizeAndValidateCreateProblemParams(input)
		if err != nil {
			issues = append(issues, service_model.BatchEntryError{Entry: index + 1, Err: serviceerr.NewValidationError("invalid problem values", err)})
			continue
		}
		normalized[index] = value
		if _, found := seenCodes[value.ClassCode]; !found {
			seenCodes[value.ClassCode] = struct{}{}
			classCodes = append(classCodes, value.ClassCode)
		}
	}

	var (
		classes []data.Class
		err     error
	)
	// Resolve all destination classes with one datastore read.
	if len(classCodes) > 0 {
		classes, err = service.Datastore.GetClassesByClassCodes(classCodes)
		if err != nil {
			return nil, nil, serviceerr.MakeGeneralError(err)
		}
	}

	// Index classes by code so each result can be mapped back to its input entry.
	classesByCode := make(map[string]data.Class, len(classes))
	for _, class := range classes {
		classesByCode[class.ClassCode] = class
	}

	// Build prepared entries and report missing classes using original entry numbers.
	prepared := make([]service_model.PreparedProblem, len(inputs))
	for index, input := range normalized {
		if input.ClassCode == "" {
			continue
		}
		class, found := classesByCode[input.ClassCode]
		if !found {
			issues = append(issues, service_model.BatchEntryError{Entry: index + 1, Err: serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("classCode %s doesn't match with any classID", input.ClassCode), nil)})
			continue
		}
		prepared[index] = service_model.PreparedProblem{Entry: index + 1, Params: input, Class: mapDataClassToServiceClass(class)}
	}

	// Return all issues together and never expose a partially prepared batch.
	if len(issues) > 0 {
		sort.SliceStable(issues, func(left, right int) bool { return issues[left].Entry < issues[right].Entry })
		return nil, issues, nil
	}
	return prepared, nil, nil
}

// ExecuteCreationBatch revalidates every entry before the first write.
func (service *problemService) ExecuteCreationBatch(batch []service_model.PreparedProblem) (service_model.ProblemCreateBatchResult, []service_model.BatchEntryError, error) {
	var result service_model.ProblemCreateBatchResult
	if err := helpers.ValidateBatchSize(len(batch)); err != nil {
		return result, []service_model.BatchEntryError{{Err: err}}, nil
	}

	inputs := make([]service_model.CreateProblemParams, len(batch))
	// Extract submitted values so they can be checked against current datastore state.
	for index, entry := range batch {
		inputs[index] = entry.Params
	}

	// Revalidate the inputs and resolve their current destination classes.
	prepared, issues, err := service.PrepareCreationBatch(inputs)
	if err != nil {
		return result, nil, serviceerr.NewBatchPreparationError("could not revalidate problem batch", err)
	}
	if len(issues) > 0 {
		return result, issues, nil
	}

	// Reject destination classes that changed identity since the preview.
	for index, entry := range prepared {
		if entry.Class.ID != batch[index].Class.ID {
			issues = append(issues, service_model.BatchEntryError{Entry: index + 1, Err: serviceerr.NewConflictError("destination class changed; validate a new batch", nil)})
		}
	}

	if len(issues) > 0 {
		return result, issues, nil
	}

	// Map the validated entries to storage values and commit them atomically.
	problems := make([]data.Problem, len(prepared))
	for index, entry := range prepared {
		problems[index] = mapCreateProblemParamsToDataProblem(entry.Params, entry.Class.ID)
	}

	problemIDs, err := service.Datastore.AddProblemsBatch(problems)
	if err != nil {
		return result, nil, serviceerr.MakeGeneralError(err)
	}
	result.CreatedProblemIDs = problemIDs
	return result, nil, nil
}

// RemoveProblemFromClass removes a problem from a class
func (service *problemService) RemoveProblemFromClass(problemID int64) error {
	if isValidProblem, err := helpers.ExistsProblem(service.Datastore, problemID); err != nil {
		return err
	} else if !isValidProblem {
		return serviceerr.NewProblemNotFoundError(fmt.Sprintf("problem %d doesn't exist", problemID), nil)
	}

	if _, err := service.Datastore.RemoveProblemByID(problemID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}

	return nil
}

// PrepareProblemReplacement validates replacement values and resolves the target without writing.
func (service *problemService) PrepareProblemReplacement(params service_model.ReplaceProblemParams) (service_model.PreparedProblemReplacement, error) {
	params, err := service_model.NormalizeAndValidateReplaceProblemParams(params)
	if err != nil {
		return service_model.PreparedProblemReplacement{}, serviceerr.NewValidationError("invalid problem values", err)
	}
	problemID := params.ProblemID
	if isValidProblem, err := helpers.ExistsProblem(service.Datastore, problemID); err != nil {
		return service_model.PreparedProblemReplacement{}, err
	} else if !isValidProblem {
		return service_model.PreparedProblemReplacement{}, serviceerr.NewProblemNotFoundError(fmt.Sprintf("problem %d doesn't exist", problemID), nil)
	}

	class, err := service.Datastore.GetClassByClassCode(params.ClassCode)
	if err != nil {
		if dataerr.IsNotFound(err) {
			return service_model.PreparedProblemReplacement{}, serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("destination class %s doesn't exist", params.ClassCode), err)
		}
		return service_model.PreparedProblemReplacement{}, serviceerr.MakeGeneralError(err)
	}
	if err := service_model.ValidateClassCodeLength(class.ClassCode); err != nil {
		return service_model.PreparedProblemReplacement{}, serviceerr.NewValidationError("invalid class code", err)
	}

	return service_model.PreparedProblemReplacement{Params: params, Class: mapDataClassToServiceClass(class)}, nil
}

// ReplaceProblemByProblemID replaces a problem's editable values.
func (service *problemService) ReplaceProblemByProblemID(params service_model.ReplaceProblemParams) error {
	prepared, err := service.PrepareProblemReplacement(params)
	if err != nil {
		return err
	}
	return service.replacePreparedProblem(prepared)
}

func (service *problemService) replacePreparedProblem(prepared service_model.PreparedProblemReplacement) error {
	params := prepared.Params
	problemID := params.ProblemID
	if err := service.Datastore.ReplaceProblemByProblemID(problemID, mapReplaceProblemParamsToDataProblem(params, prepared.Class.ID)); err != nil {
		switch {
		case dataerr.IsNotFound(err):
			// Defensive mapping in case the row disappears between pre-check and replacement.
			return serviceerr.NewProblemNotFoundError(fmt.Sprintf("problem %d doesn't exist", problemID), err)
		case dataerr.IsRowConflict(err):
			return serviceerr.NewDuplicateProblemError(
				fmt.Sprintf("duplicate problem detected while replacing problemID=%d (name=%q, classCode=%s)", problemID, params.Name, params.ClassCode),
				err,
			)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	return nil
}

// PrepareProblemPatch validates the supplied fields, verifies the target, and
// resolves a destination class only when the patch explicitly changes it.
func (service *problemService) PrepareProblemPatch(params service_model.PatchProblemParams) (service_model.PreparedProblemPatch, error) {
	params, err := service_model.NormalizeAndValidatePatchProblemParams(params)
	if err != nil {
		return service_model.PreparedProblemPatch{}, serviceerr.NewValidationError("invalid problem values", err)
	}

	if _, err := service.Datastore.GetProblemByID(params.ProblemID); err != nil {
		switch {
		case dataerr.IsNotFound(err):
			return service_model.PreparedProblemPatch{}, serviceerr.NewProblemNotFoundError(fmt.Sprintf("problem %d doesn't exist", params.ProblemID), err)
		default:
			return service_model.PreparedProblemPatch{}, serviceerr.MakeGeneralError(err)
		}
	}

	prepared := service_model.PreparedProblemPatch{Params: params}
	if !params.ClassCode.Set {
		return prepared, nil
	}

	class, err := service.Datastore.GetClassByClassCode(params.ClassCode.Value)
	if err != nil {
		if dataerr.IsNotFound(err) {
			return service_model.PreparedProblemPatch{}, serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("destination class %s doesn't exist", params.ClassCode.Value), err)
		}
		return service_model.PreparedProblemPatch{}, serviceerr.MakeGeneralError(err)
	}
	if err := service_model.ValidateClassCodeLength(class.ClassCode); err != nil {
		return service_model.PreparedProblemPatch{}, serviceerr.NewValidationError("invalid class code", err)
	}
	prepared.Class = mapDataClassToServiceClass(class)
	return prepared, nil
}

// PrepareProblemPatchBatch validates all supplied fields and resolves targets and
// explicitly requested destination classes with one read per set.
func (service *problemService) PrepareProblemPatchBatch(inputs []service_model.PatchProblemParams) ([]service_model.PreparedProblemPatch, []service_model.BatchEntryError) {
	if err := helpers.ValidateBatchSize(len(inputs)); err != nil {
		return nil, []service_model.BatchEntryError{{Err: err}}
	}

	normalized := make([]service_model.PatchProblemParams, len(inputs))
	var issues []service_model.BatchEntryError
	problemIDs := make([]int64, 0, len(inputs))
	seenIDs := make(map[int64]struct{}, len(inputs))
	for index, input := range inputs {
		value, err := service_model.NormalizeAndValidatePatchProblemParams(input)
		if err != nil {
			issues = append(issues, service_model.BatchEntryError{Entry: index + 1, Err: serviceerr.NewValidationError("invalid problem values", err)})
			continue
		}
		normalized[index] = value
		if _, found := seenIDs[value.ProblemID]; !found {
			seenIDs[value.ProblemID] = struct{}{}
			problemIDs = append(problemIDs, value.ProblemID)
		}
	}

	var (
		problems []data.Problem
		err      error
	)
	if len(problemIDs) > 0 {
		problems, err = service.Datastore.GetProblemsByIDs(problemIDs)
		if err != nil {
			for index, input := range normalized {
				if input.ProblemID > 0 {
					issues = append(issues, service_model.BatchEntryError{Entry: index + 1, Err: serviceerr.MakeGeneralError(err)})
				}
			}
			sort.SliceStable(issues, func(left, right int) bool { return issues[left].Entry < issues[right].Entry })
			return nil, issues
		}
	}

	problemsByID := make(map[int64]data.Problem, len(problems))
	for _, problem := range problems {
		problemsByID[problem.ID] = problem
	}

	classCodes := make([]string, 0, len(inputs))
	seenCodes := make(map[string]struct{}, len(inputs))
	for index, input := range normalized {
		if input.ProblemID <= 0 {
			continue
		}
		if _, found := problemsByID[input.ProblemID]; !found {
			issues = append(issues, service_model.BatchEntryError{
				Entry: index + 1,
				Err:   serviceerr.NewProblemNotFoundError(fmt.Sprintf("problem %d doesn't exist", input.ProblemID), nil),
			})
			continue
		}
		if !input.ClassCode.Set {
			continue
		}
		if _, found := seenCodes[input.ClassCode.Value]; found {
			continue
		}
		seenCodes[input.ClassCode.Value] = struct{}{}
		classCodes = append(classCodes, input.ClassCode.Value)
	}

	var classes []data.Class
	if len(classCodes) > 0 {
		classes, err = service.Datastore.GetClassesByClassCodes(classCodes)
		if err != nil {
			for index, input := range normalized {
				if !input.ClassCode.Set {
					continue
				}
				if _, exists := problemsByID[input.ProblemID]; !exists {
					continue
				}
				issues = append(issues, service_model.BatchEntryError{Entry: index + 1, Err: serviceerr.MakeGeneralError(err)})
			}
			sort.SliceStable(issues, func(left, right int) bool { return issues[left].Entry < issues[right].Entry })
			return nil, issues
		}
	}
	classesByCode := make(map[string]data.Class, len(classes))
	for _, class := range classes {
		classesByCode[class.ClassCode] = class
	}

	prepared := make([]service_model.PreparedProblemPatch, len(inputs))
	for index, input := range normalized {
		if input.ProblemID <= 0 {
			continue
		}
		if _, found := problemsByID[input.ProblemID]; !found {
			continue
		}
		entry := service_model.PreparedProblemPatch{Entry: index + 1, Params: input}
		if input.ClassCode.Set {
			class, found := classesByCode[input.ClassCode.Value]
			if !found {
				issues = append(issues, service_model.BatchEntryError{
					Entry: index + 1,
					Err:   serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("destination class %s doesn't exist", input.ClassCode.Value), nil),
				})
				continue
			}
			if err := service_model.ValidateClassCodeLength(class.ClassCode); err != nil {
				issues = append(issues, service_model.BatchEntryError{Entry: index + 1, Err: serviceerr.NewValidationError("invalid class code", err)})
				continue
			}
			entry.Class = mapDataClassToServiceClass(class)
		}
		prepared[index] = entry
	}
	if len(issues) > 0 {
		sort.SliceStable(issues, func(left, right int) bool { return issues[left].Entry < issues[right].Entry })
		return nil, issues
	}
	return prepared, nil
}

// ExecuteProblemPatchBatch revalidates every patch before applying the first write.
func (service *problemService) ExecuteProblemPatchBatch(batch []service_model.PreparedProblemPatch) (service_model.ProblemPatchBatchResult, []service_model.BatchEntryError, error) {
	var result service_model.ProblemPatchBatchResult
	if err := helpers.ValidateBatchSize(len(batch)); err != nil {
		return result, []service_model.BatchEntryError{{Err: err}}, nil
	}

	inputs := make([]service_model.PatchProblemParams, len(batch))
	for index, entry := range batch {
		inputs[index] = entry.Params
	}
	prepared, issues := service.PrepareProblemPatchBatch(inputs)
	if len(issues) > 0 {
		return result, issues, nil
	}

	for index, entry := range prepared {
		if entry.Params.ClassCode.Set && entry.Class.ID != batch[index].Class.ID {
			issues = append(issues, service_model.BatchEntryError{Entry: index + 1, Err: serviceerr.NewConflictError("destination class changed; validate a new batch", nil)})
		}
	}
	if len(issues) > 0 {
		return result, issues, nil
	}

	for index, entry := range prepared {
		if err := service.patchPreparedProblem(entry); err != nil {
			return result, nil, fmt.Errorf("could not patch problem at batch entry %d: %w", index+1, err)
		}
		result.PatchedProblemIDs = append(result.PatchedProblemIDs, entry.Params.ProblemID)
	}
	return result, nil, nil
}

// PatchProblemByProblemID changes only fields supplied in the patch.
func (service *problemService) PatchProblemByProblemID(params service_model.PatchProblemParams) error {
	prepared, err := service.PrepareProblemPatch(params)
	if err != nil {
		return err
	}
	return service.patchPreparedProblem(prepared)
}

func (service *problemService) patchPreparedProblem(prepared service_model.PreparedProblemPatch) error {
	params := prepared.Params
	var classID int64
	if params.ClassCode.Set {
		classID = prepared.Class.ID
	}
	patch := mapPatchProblemParamsToDataProblemPatch(params, classID)
	if err := service.Datastore.PatchProblemByProblemID(params.ProblemID, patch); err != nil {
		switch {
		case dataerr.IsNotFound(err):
			return serviceerr.NewProblemNotFoundError(fmt.Sprintf("problem %d doesn't exist", params.ProblemID), err)
		case dataerr.IsRowConflict(err):
			return serviceerr.NewDuplicateProblemError(
				fmt.Sprintf("duplicate problem detected while patching problemID=%d (name=%q)", params.ProblemID, params.Name.Value),
				err,
			)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}
	return nil
}

// GetProblemsByClass retrieves a class's problems ordered by ID.
func (service *problemService) GetProblemsByClass(classCode string) ([]service_model.Problem, error) {
	class, err := service.Datastore.GetClassByClassCode(classCode)
	if err != nil {
		switch {
		case dataerr.IsNotFound(err):
			return nil, serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("classCode %s doesn't match with any classID", classCode), err)
		default:
			return nil, serviceerr.MakeGeneralError(err)
		}
	}

	problemsData, err := service.Datastore.GetProblemsByClass(class.ID)
	if err != nil {
		return nil, serviceerr.MakeGeneralError(err)
	}

	// Map data.Problem to Problem
	problems := mapDataProblemsToServiceProblems(problemsData)

	helpers.SortProblemsByID(problems)
	return problems, nil
}

// GetProblemsGroupedByTopic retrieves all problems in a class grouped by topic.
func (service *problemService) GetProblemsGroupedByTopic(classCode string) ([]service_model.ProblemsByTopic, error) {
	problems, err := service.GetProblemsByClass(classCode)
	if err != nil {
		return []service_model.ProblemsByTopic{}, err
	}

	// First appearance sets each topic's position by its lowest problem ID.
	// Group problems in map
	topicMap := make(map[string][]service_model.Problem)
	var topicOrder []string

	for _, problem := range problems {
		topic := problem.GetTopic()
		if _, exists := topicMap[topic]; !exists {
			topicOrder = append(topicOrder, topic)
		}
		topicMap[topic] = append(topicMap[topic], problem)
	}

	// Convert map to []ProblemsByTopic maintaining order
	grouped := make([]service_model.ProblemsByTopic, 0, len(topicMap))
	for _, topic := range topicOrder {
		grouped = append(grouped, service_model.ProblemsByTopic{
			Topic:    topic,
			Problems: topicMap[topic],
		})
	}

	return grouped, nil
}
