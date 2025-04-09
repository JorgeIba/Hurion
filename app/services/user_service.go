package services

import (
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/validators"
)

type UserService struct {
	Datastore         data.DataStore
	DatabaseValidator *validators.ValidatorDB
}

// By UserID
func (service *UserService) DeregisterUserIDFromClass(userID string, classCode string) error {
	// Validate if user is registered in the class
	isMyBoyEnrolled, err := service.DatabaseValidator.IsEnrolledByClassCode(userID, classCode)
	if err != nil {
		return err
	} else if !isMyBoyEnrolled {
		return fmt.Errorf("user %s is not registered in class %s", userID, classCode)
	}

	// Get Info
	classID, err := service.Datastore.GetClassIDByClassCode(classCode)
	if err != nil {
		return err
	}

	// Deregister
	err = service.Datastore.DeregisterUserFromClass(userID, classID)
	return err
}

func (service *UserService) UndoACByProblemAndUserID(problemID int64, userID string) error {
	// Validate user is registered in the class of the problem
	if isBroAbleToAC, err := service.canBroACTheProblem(userID, problemID); err != nil {
		return err
	} else if !isBroAbleToAC {
		return fmt.Errorf("bro %s cannot UndoAC the problem with id: %d", userID, problemID)
	}

	// Undo-ing
	err := service.Datastore.RemoveAC(userID, problemID)
	return err
}

// By UserName
func (service *UserService) RegisterUsernameIntoClass(userName, classCode string) error {
	// Get class
	class, err := service.Datastore.GetClassByClassCode(classCode)
	if err != nil {
		return err
	}

	discordID, err := service.Datastore.GetDiscordIDByUserName(userName)
	if err != nil {
		return err
	}

	// Discord User should have binded a User Name
	isUserBinded, err := service.Datastore.IsUserBinded(discordID)
	if err != nil {
		return err
	} else if !isUserBinded {
		return fmt.Errorf("user has not username binded yet")
	}

	// Check if user was already registered
	isAlreadyRegistered, err := service.Datastore.IsUserRegisteredInClass(discordID, class.ID)
	if err != nil {
		return err
	} else if isAlreadyRegistered {
		return fmt.Errorf("user has already registered to this class")
	}

	err = service.Datastore.RegisterUserIntoClass(discordID, class.ID)
	return err
}

func (service *UserService) DeregisterUsernameFromClass(userName, classCode string) error {
	userID, err := service.Datastore.GetDiscordIDByUserName(userName)
	if err != nil {
		return err
	}

	err = service.DeregisterUserIDFromClass(userID, classCode)
	return err
}

func (service *UserService) AddACToUser(userID string, problemID int64) error {
	// Validate problem exists
	if isProblemPresent, err := service.DatabaseValidator.ExistsProblem(problemID); err != nil {
		return err
	} else if !isProblemPresent {
		return fmt.Errorf("are you sure problem %d exists?", problemID)
	}

	// Validate user is registered in the class of the problem
	if isBroAbleToAC, err := service.canBroACTheProblem(userID, problemID); err != nil {
		return err
	} else if !isBroAbleToAC {
		return fmt.Errorf("bro %s cannot AC the problem with id: %d", userID, problemID)
	}

	// ACing
	err := service.Datastore.AddAC(userID, problemID)
	return err
}

func (service *UserService) UndoACByProblemAndUsername(problemID int64, userName string) error {
	userID, err := service.Datastore.GetDiscordIDByUserName(userName)
	if err != nil {
		return err
	}

	err = service.UndoACByProblemAndUserID(problemID, userID)
	return err
}

func (service *UserService) CreateProblemIntoClass(problemName, classCode string) (int64, error) {

	if isClassCodeValid, err := service.DatabaseValidator.IsClassCodeValid(classCode); err != nil {
		return 0, fmt.Errorf("error checking if class code was valid: %v", err)
	} else if !isClassCodeValid {
		return 0, fmt.Errorf("there's no class code: %v", err)
	}

	classID, err := service.Datastore.GetClassIDByClassCode(classCode)
	if err != nil {
		return 0, err
	}

	problemID, err := service.Datastore.AddProblem(problemName, classID)
	return problemID, err
}

func (service *UserService) RemoveProblemFromClass(problemID int64) error {
	if isValidProblem, err := service.DatabaseValidator.ExistsProblem(problemID); err != nil {
		return err
	} else if !isValidProblem {
		return fmt.Errorf("are you sure problem %d exists?", problemID)
	}

	rows, err := service.Datastore.RemoveProblemByID(problemID)
	if err != nil {
		logger.Log.Error(err)
		return err
	} else if rows == 0 {
		return fmt.Errorf("i think problem id was not existing, even tho it's already removed from this universe")
	}

	return nil
}

func (service *UserService) canBroACTheProblem(userID string, problemID int64) (bool, error) {
	problem, err := service.Datastore.GetProblemByID(problemID)
	if err != nil {
		return false, err
	}

	if isMyBoyEnrolled, err := service.DatabaseValidator.IsEnrolledByClassID(userID, problem.ClassID); err != nil {
		return false, err
	} else if !isMyBoyEnrolled {
		return false, nil
	}

	return true, nil
}
