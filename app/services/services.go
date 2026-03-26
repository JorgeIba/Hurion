package services

import (
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
)

// Services aggregates all service layers
type Services struct {
	User       *UserService
	Enrollment *EnrollmentService
	Class      *ClassService
	Problem    ProblemService
	Solution   *SolutionService
	Score      *ScoreService
}

// NewServices creates a new Services instance with all service dependencies
func NewServices(datastore data.DataStore) *Services {
	// Create base services
	userService := &UserService{
		Datastore: datastore,
	}
	enrollmentService := &EnrollmentService{
		Datastore: datastore,
	}
	classService := &ClassService{
		Datastore: datastore,
	}
	problemService := NewProblemService(datastore)
	solutionService := &SolutionService{
		Datastore: datastore,
	}
	
	// Score service depends on problem service
	scoreService := &ScoreService{
		Datastore:      datastore,
		ProblemService: problemService,
	}

	return &Services{
		User:       userService,
		Enrollment: enrollmentService,
		Class:      classService,
		Problem:    problemService,
		Solution:   solutionService,
		Score:      scoreService,
	}
}
