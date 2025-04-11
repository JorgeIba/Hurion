package services

import (
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
)

type Services struct {
	User *UserService
}

func NewServices(datastore data.DataStore) *Services {
	return &Services{
		User: &UserService{
			Datastore: datastore,
		},
	}
}
