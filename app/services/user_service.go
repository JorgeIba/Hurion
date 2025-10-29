package services

import (
	"errors"
	"fmt"
	"sort"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/excelizer"
)

var (
	dataNotFoundErr = &dataerr.NotFoundError{}
	dataConflictErr = &dataerr.RowConflictError{}
)

// getDifficultyOrder returns the sort order for a difficulty
// Easy=1, Medium=2, Hard=3, Not specified=4
func getDifficultyOrder(difficulty data.Difficulty) int {
	switch difficulty {
	case data.DifficultyEasy:
		return 1
	case data.DifficultyMedium:
		return 2
	case data.DifficultyHard:
		return 3
	case data.DifficultyDefault:
		return 4
	default:
		return 5 // Unknown difficulties go last
	}
}

// sortProblemsByTopicAndDifficulty sorts problems by topic (alphabetically) and then by difficulty
func sortProblemsByTopicAndDifficulty(problems []data.Problem) {
	sort.Slice(problems, func(i, j int) bool {
		topicI := problems[i].GetTopic()
		topicJ := problems[j].GetTopic()

		// First, sort by topic (alphabetically)
		if topicI != topicJ {
			return topicI < topicJ
		}

		// If topics are the same, sort by difficulty
		diffI := problems[i].GetDifficulty()
		diffJ := problems[j].GetDifficulty()
		return getDifficultyOrder(diffI) < getDifficultyOrder(diffJ)
	})
}

type UserService struct {
	Datastore data.DataStore
}

// By UserID
func (service *UserService) DeregisterUserIDFromClass(userID string, classCode string) error {
	// Validate if user is registered in the class
	isMyBoyEnrolled, err := service.isUserIDEnrolledByClassCode(userID, classCode)
	if err != nil {
		return err
	} else if !isMyBoyEnrolled {
		return serviceerr.NewNotFoundInfoError(fmt.Sprintf("user %s is not registered in class %s", userID, classCode), nil)
	}

	// Get Info
	class, err := service.Datastore.GetClassByClassCode(classCode)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return serviceerr.NewNotFoundInfoError("", err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	// Deregister
	if err = service.Datastore.DeregisterUserFromClass(userID, class.ID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

func (service *UserService) UndoACByProblemAndUserID(problemID int64, userID string) error {
	// Validate user is registered in the class of the problem
	if isBroAbleToAC, err := service.canBroACTheProblem(userID, problemID); err != nil {
		return err
	} else if !isBroAbleToAC {
		return serviceerr.NewNotFoundInfoError(fmt.Sprintf("bro %s cannot UndoAC the problem with id: %d", userID, problemID), nil)
	}

	// Undo-ing
	if err := service.Datastore.RemoveAC(userID, problemID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

func (service *UserService) RegisterUserIDIntoClass(userID, classCode string) error {
	// Get class
	class, err := service.Datastore.GetClassByClassCode(classCode)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return serviceerr.NewNotFoundInfoError("", err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	// Discord User should have binded a User Name
	isUserBinded, err := service.Datastore.IsUserBinded(userID)
	if err != nil {
		return serviceerr.MakeGeneralError(err)
	} else if !isUserBinded {
		return serviceerr.NewNotFoundInfoError(fmt.Sprintf("user %s has not username binded yet", userID), nil)
	}

	// Check if user was already registered
	isAlreadyRegistered, err := service.Datastore.IsUserRegisteredInClass(userID, class.ID)
	if err != nil {
		return serviceerr.MakeGeneralError(err)
	} else if isAlreadyRegistered {
		return serviceerr.NewConflictError(fmt.Sprintf("user %s has already registered to this class", userID), nil)
	}

	if err = service.Datastore.RegisterUserIntoClass(userID, class.ID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

func (service *UserService) AddPendingACOfUserID(messageID, userID string, problemID int64) error {
	// Validate problem exists
	if isProblemPresent, err := service.existsProblem(problemID); err != nil {
		return err
	} else if !isProblemPresent {
		return serviceerr.NewNotFoundInfoError(fmt.Sprintf("are you sure problem %d exists?", problemID), nil)
	}

	// Validate user is registered in the class of the problem
	if isBroAbleToAC, err := service.canBroACTheProblem(userID, problemID); err != nil {
		return err
	} else if !isBroAbleToAC {
		return serviceerr.NewNotFoundInfoError(fmt.Sprintf("bro %s cannot AC the problem with id: %d", userID, problemID), nil)
	}

	if err := service.Datastore.AddPendingConfirmation(data.PendingConfirmation{
		MessageID: messageID,
		UserID:    userID,
		ProblemID: problemID,
	}); err != nil {
		return serviceerr.MakeGeneralError(err)
	}

	return nil
}

func (service *UserService) AddACToUser(userID string, problemID int64) error {
	// Validate problem exists
	if isProblemPresent, err := service.existsProblem(problemID); err != nil {
		return err
	} else if !isProblemPresent {
		return serviceerr.NewNotFoundInfoError(fmt.Sprintf("are you sure problem %d exists?", problemID), nil)
	}

	// Validate user is registered in the class of the problem
	if isBroAbleToAC, err := service.canBroACTheProblem(userID, problemID); err != nil {
		return err
	} else if !isBroAbleToAC {
		return serviceerr.NewNotFoundInfoError(fmt.Sprintf("bro %s cannot AC the problem with id: %d", userID, problemID), nil)
	}

	// ACing
	if err := service.Datastore.AddAC(userID, problemID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}

	return nil
}

func (service *UserService) GetUsernameByUserID(userID string) (string, error) {
	user, err := service.Datastore.GetUserByUserID(userID)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return "", serviceerr.NewNotFoundInfoError(fmt.Sprintf("userID %s doesn't match with any user", userID), err)
		default:
			return "", serviceerr.MakeGeneralError(err)
		}
	}
	return user.UserName, nil
}

// By UserName
func (service *UserService) RegisterUsernameIntoClass(userName, classCode string) error {
	user, err := service.Datastore.GetUserByUserName(userName)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return serviceerr.NewNotFoundInfoError("", err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	err = service.RegisterUserIDIntoClass(user.UserID, classCode)
	return err
}

func (service *UserService) DeregisterUsernameFromClass(userName, classCode string) error {
	user, err := service.Datastore.GetUserByUserName(userName)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return serviceerr.NewNotFoundInfoError("", err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	err = service.DeregisterUserIDFromClass(user.UserID, classCode)
	return err
}

func (service *UserService) AddACToUsername(userName string, problemID int64) error {
	user, err := service.Datastore.GetUserByUserName(userName)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return serviceerr.NewNotFoundInfoError("", err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	return service.AddACToUser(user.UserID, problemID)
}

func (service *UserService) UndoACByProblemAndUsername(problemID int64, userName string) error {
	user, err := service.Datastore.GetUserByUserName(userName)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return serviceerr.NewNotFoundInfoError("", err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	err = service.UndoACByProblemAndUserID(problemID, user.UserID)
	return err
}

func (service *UserService) GetUserIDByUsername(userName string) (string, error) {
	user, err := service.Datastore.GetUserByUserName(userName)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return "", serviceerr.NewNotFoundInfoError("", err)
		default:
			return "", serviceerr.MakeGeneralError(err)
		}
	}

	return user.UserID, nil
}

func (service *UserService) CreateProblemIntoClass(params CreateProblemParams) (int64, error) {
	if isClassCodeValid, err := service.isClassCodeValid(params.ClassCode); err != nil {
		return 0, err
	} else if !isClassCodeValid {
		return 0, serviceerr.NewNotFoundInfoError(fmt.Sprintf("no info of class code %s", params.ClassCode), nil)
	}

	class, err := service.Datastore.GetClassByClassCode(params.ClassCode)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return 0, serviceerr.NewNotFoundInfoError("", err)
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
		case errors.As(err, &dataConflictErr):
			return problemID, serviceerr.NewConflictError("problem with the same name in the same class", nil)
		default:
			return 0, serviceerr.MakeGeneralError(err)
		}
	}

	return problemID, nil
}

func (service *UserService) RemoveProblemFromClass(problemID int64) error {
	if isValidProblem, err := service.existsProblem(problemID); err != nil {
		return err
	} else if !isValidProblem {
		return serviceerr.NewNotFoundInfoError(fmt.Sprintf("problem %d doesn't exist", problemID), nil)
	}

	if _, err := service.Datastore.RemoveProblemByID(problemID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}

	return nil
}

func (service *UserService) DeleteClassByClassCode(classCode string) error {
	class, err := service.Datastore.GetClassByClassCode(classCode)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return serviceerr.NewNotFoundInfoError("", err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	if err = service.Datastore.ClearClass(class.ID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

func (service *UserService) CreateClass(classCode, className string) error {
	_, err := service.Datastore.CreateClass(data.Class{
		ClassCode: classCode,
		Name:      className,
	})

	if err != nil {
		switch {
		case errors.As(err, &dataConflictErr):
			return serviceerr.NewConflictError("", err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	return nil
}

func (service *UserService) BindUser(userID, userName string) error {
	// Check if the username is available for this user
	isAvailable, err := service.Datastore.IsUsernameAvailable(userName, userID)
	if err != nil {
		return serviceerr.MakeGeneralError(err)
	}

	if !isAvailable {
		return serviceerr.NewConflictError(fmt.Sprintf("username '%s' is already taken by another user", userName), nil)
	}

	// Proceed with binding the user
	if err := service.Datastore.BindUser(data.User{
		UserID:   userID,
		UserName: userName,
	}); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

func (service *UserService) GetProblemsGroupedByTopic(classCode string) ([]ProblemsByTopic, error) {
	class, err := service.Datastore.GetClassByClassCode(classCode)
	emptySlice := []ProblemsByTopic{}
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return emptySlice, serviceerr.NewNotFoundInfoError("", err)
		default:
			return emptySlice, serviceerr.MakeGeneralError(err)
		}
	}

	problems, err := service.Datastore.GetProblemsByClass(class.ID)
	if err != nil {
		return emptySlice, serviceerr.MakeGeneralError(err)
	}

	// Sort problems by topic and difficulty
	sortProblemsByTopicAndDifficulty(problems)

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

func (service *UserService) getRawScoreByClassCode(classCode string) (map[data.User][]data.SolvedProblem, error) {
	class, err := service.Datastore.GetClassByClassCode(classCode)
	emptyMap := map[data.User][]data.SolvedProblem{}
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
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

func (service *UserService) GetPendingConfirmationByMessageID(messageID string) (*data.PendingConfirmation, error) {
	userID, problemID, err := service.Datastore.GetPendingConfirmation(messageID)

	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return &data.PendingConfirmation{}, serviceerr.NewNotFoundInfoError("", err)
		default:
			return &data.PendingConfirmation{}, serviceerr.MakeGeneralError(err)
		}
	}

	return &data.PendingConfirmation{MessageID: messageID, UserID: userID, ProblemID: problemID}, nil
}

func (service *UserService) AcceptACProposal(messageID string) error {
	pendingConfirmation, err := service.GetPendingConfirmationByMessageID(messageID)
	if err != nil {
		return err
	}

	if err := service.Datastore.RemovePendingConfirmation(messageID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}

	if err := service.AddACToUser(pendingConfirmation.UserID, pendingConfirmation.ProblemID); err != nil {
		return err
	}

	return nil
}

func (service *UserService) RejectACProposal(messageID string) error {
	if err := service.Datastore.RemovePendingConfirmation(messageID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

func (service *UserService) GetListOfUsersByClassCode(classCode string) ([]data.User, error) {
	class, err := service.Datastore.GetClassByClassCode(classCode)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return []data.User{}, serviceerr.NewNotFoundInfoError(fmt.Sprintf("classCode %s doesn't match with any classID", classCode), err)
		default:
			return []data.User{}, serviceerr.MakeGeneralError(err)
		}
	}

	users, err := service.Datastore.GetAllUsersByClassID(class.ID)
	if err != nil {
		return []data.User{}, serviceerr.MakeGeneralError(err)
	}

	return users, nil
}

func (service *UserService) GetScoreByClassCode(classCode string) (Score, error) {
	solvedProblems, err := service.getRawScoreByClassCode(classCode)
	if err != nil {
		return Score{}, err
	}

	sortedSolvedProblems := sortScore(solvedProblems)

	groupedProblems, err := service.GetProblemsGroupedByTopic(classCode)
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

func (service *UserService) GetNeighborsScoreForUsername(userName, classCode string) (Score, error) {
	userID, err := service.GetUserIDByUsername(userName)
	if err != nil {
		return Score{}, err
	}

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

func (service *UserService) GetAllClasses() ([]data.Class, error) {
	classes, err := service.Datastore.GetAllClasses()
	if err != nil {
		return []data.Class{}, serviceerr.MakeGeneralError(err)
	}
	return classes, nil
}

func (service *UserService) GetScoreAllClasses() (map[string]Score, error) {
	classes, err := service.GetAllClasses()
	if err != nil {
		return nil, err
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

func (service *UserService) UpdateProblemByProblemID(problemID int64, problem data.Problem) error {
	if isValidProblem, err := service.existsProblem(problemID); err != nil {
		return err
	} else if !isValidProblem {
		return serviceerr.NewNotFoundInfoError(fmt.Sprintf("problem %d doesn't exist", problemID), nil)
	}

	if err := service.Datastore.UpdateProblemByProblemID(problemID, problem); err != nil {
		return serviceerr.MakeGeneralError(fmt.Errorf("something happened trying to update problem %d with problem: %v", problemID, problem))
	}

	return nil
}

func (service *UserService) GetClassIDByClassCode(classCode string) (int64, error) {
	class, err := service.Datastore.GetClassByClassCode(classCode)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return 0, serviceerr.NewNotFoundInfoError(fmt.Sprintf("classCode %s doesn't match with any classID", classCode), err)
		default:
			return 0, serviceerr.MakeGeneralError(err)
		}
	}

	return class.ID, nil
}

func (service *UserService) GetClassesByUserID(userID string) ([]data.Class, error) {
	classes, err := service.Datastore.GetClassesByUserID(userID)
	if err != nil {
		return []data.Class{}, serviceerr.MakeGeneralError(fmt.Errorf("something happened trying to get classes by userID: %s: %v", userID, err))
	}
	return classes, nil
}

// Private
func (service *UserService) isUserIDEnrolledByClassCode(userID, classCode string) (bool, error) {
	class, err := service.Datastore.GetClassByClassCode(classCode)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return false, serviceerr.NewNotFoundInfoError("classID doesn't match with any class", err)
		default:
			return false, serviceerr.MakeGeneralError(err)
		}
	}

	return service.isUserIDEnrolledByClassID(userID, class.ID)
}

func sortScore(solvedProblemsByUsers map[data.User][]data.SolvedProblem) []PairUserSolvedProblems {
	var userScores []PairUserSolvedProblems

	for user, problems := range solvedProblemsByUsers {
		userScores = append(userScores, PairUserSolvedProblems{
			User:           user,
			SolvedProblems: problems,
		})
	}

	sort.Slice(userScores, func(i, j int) bool {
		return len(userScores[i].SolvedProblems) > len(userScores[j].SolvedProblems)
	})

	return userScores
}

func (service *UserService) existsProblem(problemID int64) (bool, error) {
	if _, err := service.Datastore.GetProblemByID(problemID); err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return false, nil
		default:
			return false, serviceerr.MakeGeneralError(err)
		}
	}

	return true, nil
}

func (service *UserService) isClassCodeValid(classCode string) (bool, error) {
	if _, err := service.Datastore.GetClassByClassCode(classCode); err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return false, nil
		default:
			return false, serviceerr.MakeGeneralError(err)
		}
	}

	return true, nil
}

func (service *UserService) canBroACTheProblem(userID string, problemID int64) (bool, error) {
	problem, err := service.Datastore.GetProblemByID(problemID)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return false, nil
		default:
			return false, serviceerr.MakeGeneralError(err)
		}
	}

	if isMyBoyEnrolled, err := service.isUserIDEnrolledByClassID(userID, problem.ClassID); err != nil {
		return false, err
	} else if !isMyBoyEnrolled {
		return false, nil
	}

	if hasUserSolvedAlready, err := service.Datastore.HasUserSolvedProblem(userID, problemID); err != nil {
		return false, serviceerr.MakeGeneralError(fmt.Errorf("error checking if user %s has solved problem %d: %v", userID, problemID, err))
	} else if hasUserSolvedAlready {
		return false, serviceerr.NewConflictError(fmt.Sprintf("user %s has already solved problem %d", userID, problemID), nil)
	}

	return true, nil
}

func (service *UserService) isUserIDEnrolledByClassID(userID string, classID int64) (bool, error) {
	isMyBoyEnrolled, err := service.Datastore.IsUserRegisteredInClass(userID, classID)
	if err != nil {
		return false, serviceerr.NewInternalError(fmt.Sprintf("error checking if user %s is enrolled in class %d", userID, classID), err)
	}

	return isMyBoyEnrolled, nil
}

func (service *UserService) ChangeUsername(userID, newUserName string) error {
	// Check if the new username is available
	isAvailable, err := service.Datastore.IsUsernameAvailable(newUserName, userID)
	if err != nil {
		return serviceerr.MakeGeneralError(err)
	} else if !isAvailable {
		return serviceerr.NewConflictError(fmt.Sprintf("username '%s' is already taken by another user", newUserName), nil)
	}

	if err := service.Datastore.BindUser(data.User{
		UserID:   userID,
		UserName: newUserName,
	}); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

func (service *UserService) HasUsernameBound(userID string) (bool, error) {
	hasBound, err := service.Datastore.HasUsernameBound(userID)
	if err != nil {
		return false, serviceerr.MakeGeneralError(err)
	}
	return hasBound, nil
}

func (service *UserService) ExterminateUser(userID string) error {
	_, err := service.Datastore.GetUserByUserID(userID)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return serviceerr.NewNotFoundInfoError("", err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	if err = service.Datastore.ExterminateUser(userID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

// Generates an Excel file with class data (users, problems, scores)
// Returns file bytes, filename, and error
func (service *UserService) ExportClassToExcel(classCode string) ([]byte, string, error) {
	class, err := service.Datastore.GetClassByClassCode(classCode)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return nil, "", serviceerr.NewNotFoundInfoError("", err)
		default:
			return nil, "", serviceerr.MakeGeneralError(err)
		}
	}

	// Retrieve users, problems, and score
	users, err := service.Datastore.GetAllUsersByClassID(class.ID)
	if err != nil {
		return nil, "", serviceerr.MakeGeneralError(err)
	}

	score, err := service.GetScoreByClassCode(classCode)
	if err != nil {
		return nil, "", err
	}

	// Flatten problems for excelizer (maintains order from ProblemsByTopic)
	var problems []data.Problem
	for _, group := range score.ProblemsByTopic {
		problems = append(problems, group.Problems...)
	}

	// transform data layer types to excelizer types
	excelizerUsers := TransformToExcelizerUsers(users)
	excelizerProblems := TransformToExcelizerProblems(problems)
	excelizerScores := TransformToExcelizerScores(score.UserScores, problems)

	excelizer.SortUsersAlphabetically(excelizerUsers)
	excelizer.SortScoresByTotalDescending(excelizerScores)

	// Create Excel file
	excel := excelizer.CreateExcel()
	defer excel.Close()

	if err := excelizer.AddUsersSheet(excel, "Users", excelizerUsers); err != nil {
		return nil, "", serviceerr.MakeGeneralError(fmt.Errorf("error adding users sheet: %w", err))
	}

	if err := excelizer.AddProblemsSheet(excel, "Problems", excelizerProblems); err != nil {
		return nil, "", serviceerr.MakeGeneralError(fmt.Errorf("error adding problems sheet: %w", err))
	}

	if err := excelizer.AddScoresSheet(excel, "Scores", excelizerScores, excelizerProblems); err != nil {
		return nil, "", serviceerr.MakeGeneralError(fmt.Errorf("error adding scores sheet: %w", err))
	}

	// Remove default sheet
	excel.DeleteSheet("Sheet1")

	// Convert to bytes
	buffer, err := excel.WriteToBuffer()
	if err != nil {
		return nil, "", serviceerr.MakeGeneralError(fmt.Errorf("error writing excel to buffer: %w", err))
	}

	filename := fmt.Sprintf("%s_export.xlsx", classCode)
	return buffer.Bytes(), filename, nil
}
