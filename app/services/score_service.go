package services

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/services/helpers"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// ScoreService handles score calculation and ranking operations
type ScoreService struct {
	Authorizer     Authorizer
	Datastore      data.DataStore
	ProblemService ProblemService
}

// GetRawScoreByClassCode retrieves raw score data for a class
func (service *ScoreService) GetRawScoreByClassCode(classCode string) (map[service_model.User][]service_model.SolvedProblem, error) {
	class, err := service.Datastore.GetClassByClassCode(classCode)
	emptyMap := map[service_model.User][]service_model.SolvedProblem{}
	if err != nil {
		switch {
		case dataerr.IsNotFound(err):
			return emptyMap, serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("classCode %s doesn't match with any classID", classCode), err)
		default:
			return emptyMap, serviceerr.MakeGeneralError(err)
		}
	}

	studentsScoreData, err := service.Datastore.GetAllSolvedProblemsByClass(class.ID)
	if err != nil {
		return emptyMap, serviceerr.MakeGeneralError(err)
	}

	studentsScore := make(map[service_model.User][]service_model.SolvedProblem)
	for uData, spsData := range studentsScoreData {
		studentsScore[mapDataUserToServiceUser(uData)] = mapDataSolvedProblemsToServiceSolvedProblems(spsData)
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

// GetNeighborsScoreForUserID retrieves the score with neighbors around a specific user
func (service *ScoreService) GetNeighborsScoreForUserID(userID, classCode string) (service_model.Score, error) {
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
		err := serviceerr.NewUserNotEnrolledError(fmt.Sprintf("user <@%s> is not registered in class `%s`", userID, classCode), nil)
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

// GetScoreAllClasses retrieves scores for all classes in deterministic display order.
func (service *ScoreService) GetScoreAllClasses() ([]service_model.ClassScore, error) {
	classesData, err := service.Datastore.GetAllClasses()
	if err != nil {
		return nil, serviceerr.MakeGeneralError(err)
	}

	scores := make([]service_model.ClassScore, 0, len(classesData))
	for _, cData := range classesData {
		score, err := service.GetScoreByClassCode(cData.ClassCode)
		if err != nil {
			return nil, err
		}
		scores = append(scores, service_model.ClassScore{
			ClassCode: cData.ClassCode,
			ClassName: cData.Name,
			Score:     score,
		})
	}

	slices.SortFunc(scores, func(left, right service_model.ClassScore) int {
		if result := cmp.Compare(left.ClassName, right.ClassName); result != 0 {
			return result
		}

		return cmp.Compare(left.ClassCode, right.ClassCode)
	})

	return scores, nil
}

// GetClassExportData retrieves all data needed for exporting a class
// Returns users, problems grouped by topic, and score
func (service *ScoreService) GetClassExportData(actor service_model.Actor, classCode string) ([]service_model.User, []service_model.ProblemsByTopic, service_model.Score, error) {
	if err := validateAuthorizationInputs(service.Authorizer, actor); err != nil {
		return nil, nil, service_model.Score{}, err
	}

	if err := requirePermission(service.Authorizer.CanExportClass(actor)); err != nil {
		return nil, nil, service_model.Score{}, err
	}

	classData, err := service.Datastore.GetClassByClassCode(classCode)
	if err != nil {
		switch {
		case dataerr.IsNotFound(err):
			return nil, nil, service_model.Score{}, serviceerr.NewClassCodeNotFoundError(fmt.Sprintf("classCode %s doesn't match with any classID", classCode), err)
		default:
			return nil, nil, service_model.Score{}, serviceerr.MakeGeneralError(err)
		}
	}

	// Retrieve active users (those who have at least one solved problem)
	usersData, err := service.Datastore.GetActiveUsersByClassID(classData.ID)
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
