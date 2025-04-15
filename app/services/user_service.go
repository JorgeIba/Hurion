package services

import (
	"errors"
	"fmt"

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
	classID, err := service.Datastore.GetClassIDByClassCode(classCode)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return serviceerr.NewNotFoundInfoError("", err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	// Deregister
	if err = service.Datastore.DeregisterUserFromClass(userID, classID); err != nil {
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

	if err := service.Datastore.AddPendingConfirmation(messageID, userID, problemID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}

	return nil
}

// By UserName
func (service *UserService) RegisterUsernameIntoClass(userName, classCode string) error {
	userID, err := service.Datastore.GetDiscordIDByUserName(userName)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return serviceerr.NewNotFoundInfoError("", err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	err = service.RegisterUserIDIntoClass(userID, classCode)
	return err
}

func (service *UserService) DeregisterUsernameFromClass(userName, classCode string) error {
	userID, err := service.Datastore.GetDiscordIDByUserName(userName)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return serviceerr.NewNotFoundInfoError("", err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	err = service.DeregisterUserIDFromClass(userID, classCode)
	return err
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

func (service *UserService) UndoACByProblemAndUsername(problemID int64, userName string) error {
	userID, err := service.Datastore.GetDiscordIDByUserName(userName)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return serviceerr.NewNotFoundInfoError("", err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	err = service.UndoACByProblemAndUserID(problemID, userID)
	return err
}

func (service *UserService) CreateProblemIntoClass(problemName, classCode string) (int64, error) {
	if isClassCodeValid, err := service.isClassCodeValid(classCode); err != nil {
		return 0, err
	} else if !isClassCodeValid {
		return 0, serviceerr.NewNotFoundInfoError(fmt.Sprintf("no info of class code %s", classCode), nil)
	}

	classID, err := service.Datastore.GetClassIDByClassCode(classCode)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return 0, serviceerr.NewNotFoundInfoError("", err)
		default:
			return 0, serviceerr.MakeGeneralError(err)
		}
	}

	problemID, err := service.Datastore.AddProblem(problemName, classID)
	if err != nil {
		return 0, serviceerr.MakeGeneralError(err)
	} else if problemID == 0 {
		return 0, serviceerr.NewConflictError("problem with the same name in the same class", nil)
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
	classID, err := service.Datastore.GetClassIDByClassCode(classCode)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return serviceerr.NewNotFoundInfoError("", err)
		default:
			return serviceerr.MakeGeneralError(err)
		}
	}

	if err = service.Datastore.ClearClass(classID); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

func (service *UserService) CreateClass(classCode, className string) error {
	_, err := service.Datastore.CreateClass(classCode, className)

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
	if err := service.Datastore.BindUser(userID, userName); err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return nil
}

func (service *UserService) GetListOfProblemsByClassCode(classCode string) ([]*data.Problem, error) {
	classID, err := service.Datastore.GetClassIDByClassCode(classCode)
	emptySlice := []*data.Problem{}
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return emptySlice, serviceerr.NewNotFoundInfoError("", err)
		default:
			return emptySlice, serviceerr.MakeGeneralError(err)
		}
	}

	problems, err := service.Datastore.GetProblemsByClass(classID)
	if err != nil {
		return emptySlice, serviceerr.MakeGeneralError(err)
	}
	return problems, nil
}

func (service *UserService) GetScoreByClassCode(classCode string) (map[*data.User][]*data.SolvedProblem, error) {
	classID, err := service.Datastore.GetClassIDByClassCode(classCode)
	emptyMap := map[*data.User][]*data.SolvedProblem{}
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return emptyMap, serviceerr.NewNotFoundInfoError("", err)
		default:
			return emptyMap, serviceerr.MakeGeneralError(err)
		}
	}

	studentsScore, err := service.Datastore.GetAllSolvedProblemsByClass(classID)
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

func (service *UserService) GetUsersInClassCode(classCode string) ([]*data.User, error) {
	classID, err := service.Datastore.GetClassIDByClassCode(classCode)

	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return []*data.User{}, serviceerr.NewNotFoundInfoError(fmt.Sprintf("classCode %s doesn't match with classID", classCode), err)
		default:
			return []*data.User{}, serviceerr.MakeGeneralError(err)
		}
	}

	users, err := service.Datastore.GetAllUsersByClassID(classID)
	if err != nil {
		return []*data.User{}, serviceerr.MakeGeneralError(err)
	}

	return users, nil
}

// Private
func (service *UserService) isUserIDEnrolledByClassCode(userID, classCode string) (bool, error) {
	classID, err := service.Datastore.GetClassIDByClassCode(classCode)
	if err != nil {
		switch {
		case errors.As(err, &dataNotFoundErr):
			return false, serviceerr.NewNotFoundInfoError("classID doesn't match with any class", err)
		default:
			return false, serviceerr.MakeGeneralError(err)
		}
	}

	return service.isUserIDEnrolledByClassID(userID, classID)
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
	if _, err := service.Datastore.GetClassIDByClassCode(classCode); err != nil {
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

	return true, nil
}

func (service *UserService) isUserIDEnrolledByClassID(userID string, classID int64) (bool, error) {
	isMyBoyEnrolled, err := service.Datastore.IsUserRegisteredInClass(userID, classID)
	if err != nil {
		return false, serviceerr.NewInternalError(fmt.Sprintf("error checking if user %s is enrolled in class %d", userID, classID), err)
	}

	return isMyBoyEnrolled, nil
}
