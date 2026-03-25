package services

import (
	"errors"
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/helpers"
)

var (
	dataNotFoundErrProblem = &dataerr.NotFoundError{}
	dataConflictErrProblem = &dataerr.RowConflictError{}
)

// ProblemService handles problem management operations
type ProblemService struct {
	Datastore data.DataStore
}

// CreateProblemIntoClass creates a new problem in a class
func (service *ProblemService) CreateProblemIntoClass(params CreateProblemParams) (int64, error) {
	if isClassCodeValid, err := helpers.IsClassCodeValid(service.Datastore, params.ClassCode); err != nil {
		return 0, err
	} else if !isClassCodeValid {
		return 0, serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("no info of class code %s", params.ClassCode), nil)
	}

	class, err := service.Datastore.GetClassByClassCode(params.ClassCode)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErrProblem):
			return 0, serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("classCode %s doesn't match with any classID", params.ClassCode), err)
		default:
			return 0, serviceerr.MakeGeneralError(err)
		}
	}

	problemID, err := service.Datastore.AddProblem(data.Problem{
		Name:       params.Name,
		URL:        params.URL,
		ClassID:    class.ID,
		Difficulty: params.Difficulty,
		Topic:      params.Topic,
	})
	if err != nil {
		switch {
		case errors.As(err, &dataConflictErrProblem):
			return problemID, serviceerr.NewDuplicateProblemURLError("problem with the same name in the same class", nil)
		default:
			return 0, serviceerr.MakeGeneralError(err)
		}
	}

	return problemID, nil
}

// RemoveProblemFromClass removes a problem from a class
func (service *ProblemService) RemoveProblemFromClass(problemID int64) error {
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
func (service *ProblemService) UpdateProblemByProblemID(problemID int64, problem data.Problem) error {
	if isValidProblem, err := helpers.ExistsProblem(service.Datastore, problemID); err != nil {
		return err
	} else if !isValidProblem {
		return serviceerr.NewProblemNotFoundError(fmt.Sprintf("problem %d doesn't exist", problemID), nil)
	}

	if err := service.Datastore.UpdateProblemByProblemID(problemID, problem); err != nil {
		switch {
		case errors.As(err, &dataNotFoundErrProblem):
			// Defensive mapping in case the row disappears between pre-check and update.
			return serviceerr.NewProblemNotFoundError(fmt.Sprintf("problem %d doesn't exist", problemID), err)
		case errors.As(err, &dataConflictErrProblem):
			return serviceerr.NewDuplicateProblemURLError(
				fmt.Sprintf("duplicate problem detected while updating problemID=%d (name=%q, classID=%d)", problemID, problem.Name, problem.ClassID),
				err,
			)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	return nil
}

// GetProblemsGroupedByTopic retrieves all problems in a class grouped by topic
func (service *ProblemService) GetProblemsGroupedByTopic(classCode string) ([]ProblemsByTopic, error) {
	class, err := service.Datastore.GetClassByClassCode(classCode)
	emptySlice := []ProblemsByTopic{}
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErrProblem):
			return emptySlice, serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("classCode %s doesn't match with any classID", classCode), err)
		default:
			return emptySlice, serviceerr.MakeGeneralError(err)
		}
	}

	problems, err := service.Datastore.GetProblemsByClass(class.ID)
	if err != nil {
		return emptySlice, serviceerr.MakeGeneralError(err)
	}

	// Sort problems by topic and difficulty
	helpers.SortProblemsByTopicAndDifficulty(problems)

	// Group problems in map
	topicMap := make(map[string][]data.Problem)
	var topicOrder []string

	for _, problem := range problems {
		topic := problem.GetTopic()
		if _, exists := topicMap[topic]; !exists {
			topicOrder = append(topicOrder, topic)
		}
		topicMap[topic] = append(topicMap[topic], problem)
	}

	// Convert map to []ProblemsByTopic maintaining order
	grouped := make([]ProblemsByTopic, 0, len(topicMap))
	for _, topic := range topicOrder {
		grouped = append(grouped, ProblemsByTopic{
			Topic:    topic,
			Problems: topicMap[topic],
		})
	}

	return grouped, nil
}
