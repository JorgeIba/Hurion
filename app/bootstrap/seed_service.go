package bootstrap

import (
	"github.com/JorgeIba/Hurion/app/authorization"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

type seedAuthorizer struct{}

func (seedAuthorizer) Authorize(actor authorization.Actor, permission authorization.Permission, _ authorization.Resource) error {
	if actor.UserID == "bootstrap" && actor.GuildID == "bootstrap" && permission == authorization.Admin {
		return nil
	}
	return authorization.ErrDenied
}

type seedClassService struct{ *services.ClassService }

func seedActor() authorization.Actor {
	return authorization.Actor{GuildID: "bootstrap", UserID: "bootstrap"}
}
func (s seedClassService) CreateClass(code, name string) (int64, error) {
	return s.ClassService.CreateClass(seedActor(), code, name)
}
func (s seedClassService) RegisterClassroom(room service_model.Classroom) error {
	return s.ClassService.RegisterClassroom(seedActor(), room)
}
func (s seedClassService) AssignAccessRole(role service_model.ClassAccessRole) error {
	return s.ClassService.AssignAccessRole(seedActor(), role)
}

// NewDatabaseReconciler creates an isolated, trusted startup path. Its authorizer
// is never injected into services used by Discord events or scripts.
func NewDatabaseReconciler(store data.DataStore) *Reconciler {
	return NewReconciler(seedClassService{services.NewClassService(store, seedAuthorizer{})})
}
