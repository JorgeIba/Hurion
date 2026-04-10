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
	dataNotFoundErrScore = &dataerr.NotFoundError{}
)

// ScoreService handles score calculation and ranking operations
type ScoreService struct {
	Datastore      data.DataStore
	ProblemService ProblemService
}

// GetRawScoreByClassCode retrieves raw score data for a class
func (service *ScoreService) GetRawScoreByClassCode(classCode string) (map[service_model.User][]service_model.SolvedProblem, error) {
	class, err := service.Datastore.GetClassByClassCode(classCode)
	emptyMap := map[service_model.User][]service_model.SolvedProblem{}
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErrScore):
			return emptyMap, serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("classCode %s doesn't match with any classID", classCode), err)
		default:
			return emptyMap, serviceerr.MakeGeneralError(err)
		}
	}

	studentsScoreData, err := service.Datastore.GetAllSolvedProblemsByClass(class.ID)
	if err != nil {
		return emptyMap, serviceerr.MakeGeneralError(err)
	}

	// Ensure all users in the class are represented in the score map
	allUsersData, err := service.Datastore.GetAllUsersByClassID(class.ID)
	if err != nil {
		return emptyMap, serviceerr.MakeGeneralError(err)
	}

	studentsScore := make(map[service_model.User][]service_model.SolvedProblem)
	for uData, spsData := range studentsScoreData {
		studentsScore[mapDataUserToServiceUser(uData)] = mapDataSolvedProblemsToServiceSolvedProblems(spsData)
	}

	for _, uData := range allUsersData {
		user := mapDataUserToServiceUser(uData)
		if _, exists := studentsScore[user]; !exists {
			studentsScore[user] = []service_model.SolvedProblem{}
		}
	}

	return studentsScore, nil
}

// GetScoreByClassCode retrieves the full score for a class
func (service *ScoreService) GetScoreByClassCode(classCode string) (service_model.Score, error) {
	if service.ProblemService == nil {
		return service_model.Score{}, serviceerr.NewInternalError("ProblemService is not injected in ScoreService", nil)
	}

	solvedProblems, err := service.GetRawScoreByClassCode(classCode)
	if err != nil {
		return service_model.Score{}, err
	}

	groupedProblems, err := service.ProblemService.GetProblemsGroupedByTopic(classCode)
	if err != nil {
		return service_model.Score{}, err
	}

	// Build all problems list from grouped (flattened and ordered)
	allProblems := []service_model.Problem{}
	for _, group := range groupedProblems {
		allProblems = append(allProblems, group.Problems...)
	}

	allScore := make([]service_model.PairUserProblemScore, 0)

	for user, solvedProblemsByUser := range solvedProblems {
		var scoreUser service_model.PairUserProblemScore
		scoreUser.User = user
		scoreUser.Problems = make(map[int64]bool)

		// Set solved status
		problemIDContainerSolvedByUser := make(map[int64]struct{})
		for _, problemUser := range solvedProblemsByUser {
			problemIDContainerSolvedByUser[problemUser.ProblemID] = struct{}{}
		}

		for _, problem := range allProblems {
			if _, exists := problemIDContainerSolvedByUser[problem.ID]; exists {
				scoreUser.Problems[problem.ID] = true
				scoreUser.CountProblems++
			} else {
				scoreUser.Problems[problem.ID] = false
			}
		}

		allScore = append(allScore, scoreUser)
	}

	helpers.SortScore(allScore)

	return service_model.Score{
		UserScores:      allScore,
		ProblemsByTopic: groupedProblems,
	}, nil
}

// GetNeighborsScoreForUsername retrieves the score with neighbors around a specific user
func (service *ScoreService) GetNeighborsScoreForUsername(userName, classCode string) (service_model.Score, error) {
	// Get userID from username
	user, err := service.Datastore.GetUserByUserName(userName)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErrScore):
			return service_model.Score{}, serviceerr.NewUserNotFoundError(fmt.Sprintf("username `%s` not found", userName), err)
		default:
			return service_model.Score{}, serviceerr.MakeGeneralError(err)
		}
	}
	userID := user.UserID

	score, err := service.GetScoreByClassCode(classCode)
	if err != nil {
		return service_model.Score{}, err
	}

	// Find the user's position
	var userIndex int = -1
	for i, scoreUser := range score.UserScores {
		if scoreUser.User.UserID == userID {
			userIndex = i
			break
		}
	}

	if userIndex == -1 {
		err := serviceerr.NewUserNotEnrolledError(fmt.Sprintf("user `%s` is not registered in class `%s`", userName, classCode), nil)
		return service_model.Score{}, err
	}

	// Get neighbors (2 before, user, 2 after)
	l, r := max(0, userIndex-2), min(len(score.UserScores)-1, userIndex+2)
	neighborScores := score.UserScores[l : r+1]

	// Return Score with filtered UserScores but same ProblemsByTopic
	return service_model.Score{
		UserScores:      neighborScores,
		ProblemsByTopic: score.ProblemsByTopic,
	}, nil
}

// GetScoreAllClasses retrieves scores for all classes
func (service *ScoreService) GetScoreAllClasses() (map[string]service_model.Score, error) {
	classesData, err := service.Datastore.GetAllClasses()
	if err != nil {
		return nil, serviceerr.MakeGeneralError(err)
	}

	scores := make(map[string]service_model.Score)
	for _, cData := range classesData {
		score, err := service.GetScoreByClassCode(cData.ClassCode)
		if err != nil {
			return nil, err
		}
		scores[cData.Name] = score
	}

	return scores, nil
}

// GetClassExportData retrieves all data needed for exporting a class
// Returns users, problems grouped by topic, and score
func (service *ScoreService) GetClassExportData(classCode string) ([]service_model.User, []service_model.ProblemsByTopic, service_model.Score, error) {
	classData, err := service.Datastore.GetClassByClassCode(classCode)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErrScore):
			return nil, nil, service_model.Score{}, serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("classCode %s doesn't match with any classID", classCode), err)
		default:
			return nil, nil, service_model.Score{}, serviceerr.MakeGeneralError(err)
		}
	}

	// Retrieve users
	usersData, err := service.Datastore.GetAllUsersByClassID(classData.ID)
	if err != nil {
		return nil, nil, service_model.Score{}, serviceerr.MakeGeneralError(err)
	}

	// Retrieve score (contains ProblemsByTopic)
	score, err := service.GetScoreByClassCode(classCode)
	if err != nil {
		return nil, nil, service_model.Score{}, err
	}

	return mapDataUsersToServiceUsers(usersData), score.ProblemsByTopic, score, nil
}
