package services

import "github.com/JorgeIba/Hurion/app/services/service_model"

// Authorizer checks permissions for application operations.
//
//go:generate go tool mockgen -destination=mock_services/mock_authorizer.go -package=mock_services . Authorizer
type Authorizer interface {
	CanCreateClass(actor service_model.Actor) (bool, error)
	CanLinkClassResources(actor service_model.Actor) (bool, error)
	CanClearClass(actor service_model.Actor) (bool, error)
	CanDeleteClass(actor service_model.Actor) (bool, error)
	CanUnlinkClassResources(actor service_model.Actor) (bool, error)
	CanDeleteUser(actor service_model.Actor) (bool, error)
	CanExportClass(actor service_model.Actor) (bool, error)
	CanManageClasses(actor service_model.Actor, classCodes ...string) (bool, error)
}
