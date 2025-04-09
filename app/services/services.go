package services

import (
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/validators"
)

type Services struct {
	User    *UserService
	Problem *ProblemService
}

func NewServices(datastore data.DataStore, dbValidator *validators.ValidatorDB) (*Services, error) {
	return &Services{
		User: &UserService{
			Datastore:         datastore,
			DatabaseValidator: dbValidator,
		},
	}, nil
}
