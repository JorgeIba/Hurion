package services

import (
	"github.com/JorgeIba/Hurion/app/authorization"
	"github.com/JorgeIba/Hurion/app/data"
)

// Services aggregates all service layers
type Services struct {
	authorizer authorization.Authorizer
	User       *UserService
	Enrollment *EnrollmentService
	Class      *ClassService
	Problem    ProblemService
	Solution   *SolutionService
	Score      *ScoreService
}

// NewServices creates a new Services instance with all service dependencies
func NewServices(datastore data.DataStore, authorizer authorization.Authorizer) *Services {
	// Create base services
	userService := &UserService{
		Datastore:  datastore,
		Authorizer: authorizer,
	}
	enrollmentService := &EnrollmentService{
		Authorizer: authorizer,
		Datastore:  datastore,
	}
	classService := &ClassService{
		Datastore:  datastore,
		Authorizer: authorizer,
	}
	problemService := NewProblemService(datastore, authorizer)
	solutionService := &SolutionService{
		Datastore:  datastore,
		Authorizer: authorizer,
	}

	// Score service depends on problem service
	scoreService := &ScoreService{
		Datastore:      datastore,
		Authorizer:     authorizer,
		ProblemService: problemService,
	}

	return &Services{
		authorizer: authorizer,
		User:       userService,
		Enrollment: enrollmentService,
		Class:      classService,
		Problem:    problemService,
		Solution:   solutionService,
		Score:      scoreService,
	}
}
