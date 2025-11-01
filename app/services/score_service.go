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
	dataNotFoundErrScore = &dataerr.NotFoundError{}
)

// ScoreService handles score calculation and ranking operations
type ScoreService struct {
	Datastore      data.DataStore
	ProblemService *ProblemService
}

// GetRawScoreByClassCode retrieves raw score data for a class
func (service *ScoreService) GetRawScoreByClassCode(classCode string) (map[data.User][]data.SolvedProblem, error) {
	class, err := service.Datastore.GetClassByClassCode(classCode)
	emptyMap := map[data.User][]data.SolvedProblem{}
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErrScore):
			return emptyMap, serviceerr.NewNotFoundInfoError("", err)
		default:
			return emptyMap, serviceerr.MakeGeneralError(err)
		}
	}

	studentsScore, err := service.Datastore.GetAllSolvedProblemsByClass(class.ID)
	if err != nil {
		return emptyMap, serviceerr.MakeGeneralError(err)
	}

	return studentsScore, nil
}

// GetScoreByClassCode retrieves the full score for a class
func (service *ScoreService) GetScoreByClassCode(classCode string) (Score, error) {
	solvedProblems, err := service.GetRawScoreByClassCode(classCode)
	if err != nil {
		return Score{}, err
	}

	sortedSolvedProblems := helpers.SortScore(solvedProblems)

	groupedProblems, err := service.ProblemService.GetProblemsGroupedByTopic(classCode)
	if err != nil {
		return Score{}, err
	}

	// Build all problems list from grouped (flattened and ordered)
	allProblems := []data.Problem{}
	for _, group := range groupedProblems {
		allProblems = append(allProblems, group.Problems...)
	}

	allScore := make([]PairUserProblemScore, 0)

	for _, entry := range sortedSolvedProblems {
		var scoreUser PairUserProblemScore
		scoreUser.User = entry.User
		scoreUser.Problems = make(map[data.Problem]bool)

		solvedProblemsByUser := entry.SolvedProblems

		// Set
		problemIDContainerSolvedByUser := make(map[int64]struct{})
		for _, problemUser := range solvedProblemsByUser {
			problemIDContainerSolvedByUser[problemUser.ProblemID] = struct{}{}
		}

		for _, problem := range allProblems {
			if _, exists := problemIDContainerSolvedByUser[problem.ID]; exists {
				scoreUser.Problems[problem] = true
				scoreUser.CountProblems++
			} else {
				scoreUser.Problems[problem] = false
			}
		}

		allScore = append(allScore, scoreUser)
	}

	return Score{
		UserScores:      allScore,
		ProblemsByTopic: groupedProblems,
	}, nil
}

// GetNeighborsScoreForUsername retrieves the score with neighbors around a specific user
func (service *ScoreService) GetNeighborsScoreForUsername(userName, classCode string) (Score, error) {
	// Get userID from username
	user, err := service.Datastore.GetUserByUserName(userName)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErrScore):
			return Score{}, serviceerr.NewNotFoundInfoError("", err)
		default:
			return Score{}, serviceerr.MakeGeneralError(err)
		}
	}
	userID := user.UserID

	score, err := service.GetScoreByClassCode(classCode)
	if err != nil {
		return Score{}, err
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
		err := serviceerr.NewNotFoundInfoError(fmt.Sprintf("user `%s` doesn't have any AC in class `%s`", userName, classCode), nil)
		return Score{}, err
	}

	// Get neighbors (2 before, user, 2 after)
	l, r := max(0, userIndex-2), min(len(score.UserScores)-1, userIndex+2)
	neighborScores := score.UserScores[l : r+1]

	// Return Score with filtered UserScores but same ProblemsByTopic
	return Score{
		UserScores:      neighborScores,
		ProblemsByTopic: score.ProblemsByTopic,
	}, nil
}

// GetScoreAllClasses retrieves scores for all classes
func (service *ScoreService) GetScoreAllClasses() (map[string]Score, error) {
	classes, err := service.Datastore.GetAllClasses()
	if err != nil {
		return nil, serviceerr.MakeGeneralError(err)
	}

	scores := make(map[string]Score)
	for _, class := range classes {
		score, err := service.GetScoreByClassCode(class.ClassCode)
		if err != nil {
			return nil, err
		}
		scores[class.Name] = score
	}

	return scores, nil
}

// GetClassExportData retrieves all data needed for exporting a class
// Returns users, problems grouped by topic, and score
func (service *ScoreService) GetClassExportData(classCode string) ([]data.User, []ProblemsByTopic, Score, error) {
	class, err := service.Datastore.GetClassByClassCode(classCode)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErrScore):
			return nil, nil, Score{}, serviceerr.NewNotFoundInfoError("", err)
		default:
			return nil, nil, Score{}, serviceerr.MakeGeneralError(err)
		}
	}

	// Retrieve users
	users, err := service.Datastore.GetAllUsersByClassID(class.ID)
	if err != nil {
		return nil, nil, Score{}, serviceerr.MakeGeneralError(err)
	}

	// Retrieve score (contains ProblemsByTopic)
	score, err := service.GetScoreByClassCode(classCode)
	if err != nil {
		return nil, nil, Score{}, err
	}

	return users, score.ProblemsByTopic, score, nil
}
