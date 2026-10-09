package services

import (
	"fmt"

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
	CreateProblemIntoClass(params service_model.CreateProblemParams) (int64, error)
	RemoveProblemFromClass(problemID int64) error
	UpdateProblemByProblemID(params service_model.UpdateProblemParams) error
	GetProblemsGroupedByTopic(classCode string) ([]service_model.ProblemsByTopic, error)
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

// UpdateProblemByProblemID updates a problem's information
func (service *problemService) UpdateProblemByProblemID(params service_model.UpdateProblemParams) error {
	params, err := service_model.NormalizeAndValidateUpdateProblemParams(params)
	if err != nil {
		return serviceerr.NewValidationError("invalid problem values", err)
	}
	problemID := params.ProblemID
	if isValidProblem, err := helpers.ExistsProblem(service.Datastore, problemID); err != nil {
		return err
	} else if !isValidProblem {
		return serviceerr.NewProblemNotFoundError(fmt.Sprintf("problem %d doesn't exist", problemID), nil)
	}

	class, err := service.Datastore.GetClassByClassCode(params.ClassCode)
	if err != nil {
		if dataerr.IsNotFound(err) {
			return serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("destination class %s doesn't exist", params.ClassCode), err)
		}
		return serviceerr.MakeGeneralError(err)
	}
	if err := service_model.ValidateClassCodeLength(class.ClassCode); err != nil {
		return serviceerr.NewValidationError("invalid class code", err)
	}

	if err := service.Datastore.UpdateProblemByProblemID(problemID, mapUpdateProblemParamsToDataProblem(params, class.ID)); err != nil {
		switch {
		case dataerr.IsNotFound(err):
			// Defensive mapping in case the row disappears between pre-check and update.
			return serviceerr.NewProblemNotFoundError(fmt.Sprintf("problem %d doesn't exist", problemID), err)
		case dataerr.IsRowConflict(err):
			return serviceerr.NewDuplicateProblemError(
				fmt.Sprintf("duplicate problem detected while updating problemID=%d (name=%q, classCode=%s)", problemID, params.Name, params.ClassCode),
				err,
			)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	return nil
}

// GetProblemsGroupedByTopic retrieves all problems in a class grouped by topic
func (service *problemService) GetProblemsGroupedByTopic(classCode string) ([]service_model.ProblemsByTopic, error) {

	class, err := service.Datastore.GetClassByClassCode(classCode)
	emptySlice := []service_model.ProblemsByTopic{}
	if err != nil {
		switch {
		case dataerr.IsNotFound(err):
			return emptySlice, serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("classCode %s doesn't match with any classID", classCode), err)
		default:
			return emptySlice, serviceerr.MakeGeneralError(err)
		}
	}

	problemsData, err := service.Datastore.GetProblemsByClass(class.ID)
	if err != nil {
		return emptySlice, serviceerr.MakeGeneralError(err)
	}

	// Map data.Problem to Problem
	problems := mapDataProblemsToServiceProblems(problemsData)

	// First appearance sets each topic's position by its lowest problem ID.
	helpers.SortProblemsByID(problems)

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
