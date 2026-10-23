package mentor

import (
	"github.com/JorgeIba/Hurion/app/authorization/authtest"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/services"
)

func withTestAuthorization(s *services.Services, checker authtest.RoleChecker) *services.Services {
	if checker == nil {
		return s
	}
	var store data.DataStore
	if s != nil && s.Solution != nil {
		store = s.Solution.Datastore
	}
	return services.NewServices(store, authtest.Roles{Checker: checker})
}

func withTestInteractionAuthorization(s *services.Services, checker authtest.RoleChecker) *services.Services {
	if checker == nil {
		return s
	}
	var store data.DataStore
	if s != nil && s.Solution != nil {
		store = s.Solution.Datastore
	}
	return services.NewServices(store, authtest.IntakeRoles{Checker: checker})
}
