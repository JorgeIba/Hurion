package services

import (
	"github.com/JorgeIba/Hurion/app/data"
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
func NewServices(datastore data.DataStore, auth Authorizer) *Services {
	// Create base services
	userService := &UserService{
		Authorizer: auth,
		Datastore:  datastore,
	}
	enrollmentService := &EnrollmentService{
		Datastore: datastore,
	}
	classService := &ClassService{
		Authorizer: auth,
		Datastore:  datastore,
	}
	problemService := NewProblemService(datastore, auth)
	solutionService := &SolutionService{
		Authorizer: auth,
		Datastore:  datastore,
	}

	// Score service depends on problem service
	scoreService := &ScoreService{
		Authorizer:     auth,
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
