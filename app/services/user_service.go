package services

import (
	"errors"
	"fmt"
	"sort"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
)

var (
	dataNotFoundErr = &dataerr.NotFoundError{}
	dataConflictErr = &dataerr.RowConflictError{}
)

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

func (service *UserService) CreateProblemIntoClass(problemName, url, classCode string) (int64, error) {
	if isClassCodeValid, err := service.isClassCodeValid(classCode); err != nil {
		return 0, err
	} else if !isClassCodeValid {
		return 0, serviceerr.NewNotFoundInfoError(fmt.Sprintf("no info of class code %s", classCode), nil)
	}

	class, err := service.Datastore.GetClassByClassCode(classCode)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return 0, serviceerr.NewNotFoundInfoError("", err)
		default:
			return 0, serviceerr.MakeGeneralError(err)
		}
	}

	problemID, err := service.Datastore.AddProblem(data.Problem{
		Name:    problemName,
		URL:     url,
		ClassID: class.ID,
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

func (service *UserService) GetListOfProblemsByClassCode(classCode string) ([]data.Problem, error) {
	class, err := service.Datastore.GetClassByClassCode(classCode)
	emptySlice := []data.Problem{}
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

	return problems, nil
}

func (service *UserService) GetScoreByClassCode(classCode string) (map[data.User][]data.SolvedProblem, error) {
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

type PairUserSolvedProblems struct {
	User           data.User
	SolvedProblems []data.SolvedProblem
}

type PairUserProblemScore struct {
	User          data.User
	CountProblems int
	Problems      map[data.Problem]bool
}

func (service *UserService) GetSortedScoreByClassCode(classCode string) ([]PairUserProblemScore, error) {
	solvedProblems, err := service.GetScoreByClassCode(classCode)
	if err != nil {
		return []PairUserProblemScore{}, err
	}

	sortedSolvedProblems := sortScore(solvedProblems)

	allProblems, err := service.GetListOfProblemsByClassCode(classCode)
	if err != nil {
		return []PairUserProblemScore{}, err
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
	return allScore, nil
}

type TableScore struct {
	Class  string
	Header []string
	Table  [][]string
}

func (service *UserService) GetSortedScoreByClassCodeAsTable(classCode string) (TableScore, error) {
	score, err := service.GetSortedScoreByClassCode(classCode)
	if err != nil {
		return TableScore{}, err
	}

	allProblems, err := service.GetListOfProblemsByClassCode(classCode)
	if err != nil {
		return TableScore{}, err
	}

	headers := []string{"Problem \\ Username"}
	for _, entry := range score {
		headers = append(headers, fmt.Sprintf("%s (%d)", entry.User.UserName, entry.CountProblems))
	}

	var table [][]string
	for _, problem := range allProblems {
		row := []string{problem.Name}
		for _, entry := range score {
			if v, exists := entry.Problems[problem]; exists && v {
				row = append(row, "AC")
			} else {
				row = append(row, " ")
			}
		}
		table = append(table, row)
	}
	return TableScore{Header: headers, Table: table}, nil
}

func (service *UserService) GetNeighborsScoreTableForUsername(userName, classCode string) (TableScore, error) {
	userID, err := service.GetUserIDByUsername(userName)
	if err != nil {
		return TableScore{}, err
	}

	score, err := service.GetSortedScoreByClassCode(classCode)
	if err != nil {
		return TableScore{}, err
	}

	// Find the user's position
	var userIndex int
	for i, scoreUser := range score {
		if scoreUser.User.UserID == userID {
			userIndex = i
			break
		}
	}

	if userIndex == -1 {
		err := serviceerr.NewNotFoundInfoError(fmt.Sprintf("user `%s` doesn't have any AC in class `%s`", userName, classCode), nil)
		return TableScore{}, err
	}

	l, r := max(0, userIndex-2), min(len(score)-1, userIndex+2)
	score = score[l : r+1]

	allProblems, err := service.GetListOfProblemsByClassCode(classCode)
	if err != nil {
		return TableScore{}, err
	}

	headers := []string{"Problem \\ Username"}
	for _, entry := range score {
		headers = append(headers, fmt.Sprintf("%s (%d)", entry.User.UserName, entry.CountProblems))
	}

	var table [][]string

	for _, problem := range allProblems {
		row := []string{problem.Name}
		for _, entry := range score {
			if v, exists := entry.Problems[problem]; exists && v {
				row = append(row, "AC")
			} else {
				row = append(row, " ")
			}
		}

		table = append(table, row)
	}
	return TableScore{Header: headers, Table: table}, nil
}

func (service *UserService) GetAllClasses() ([]data.Class, error) {
	classes, err := service.Datastore.GetAllClasses()
	if err != nil {
		return []data.Class{}, serviceerr.MakeGeneralError(err)
	}
	return classes, nil
}

func (service *UserService) GetScoreAllClasses() ([]TableScore, error) {
	classes, err := service.GetAllClasses()
	if err != nil {
		return []TableScore{}, err
	}

	var scores []TableScore
	for _, class := range classes {
		scoreTable, err := service.GetSortedScoreByClassCodeAsTable(class.ClassCode)
		if err != nil {
			return []TableScore{}, err
		}
		scoreTable.Class = class.Name
		scores = append(scores, scoreTable)
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
