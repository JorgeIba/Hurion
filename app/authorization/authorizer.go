package authorization

import (
	"maps"
	"slices"
	"strings"

	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// RoleSource supplies current role IDs without interpreting their permissions.
//
//go:generate go tool mockgen -destination=mock_authorization/mock_role_source.go -package=mock_authorization . RoleSource
type RoleSource interface {
	FetchRoles(userID string) ([]string, error)
}

// Authorizer grants operation permissions to users and trusted processes.
// The configuration reconciler may only create classes and link configuration.
type Authorizer struct {
	roleSource          RoleSource
	adminRoleID         string
	mentorRoleID        string
	mentorRoleOverrides map[string]string
}

var _ services.Authorizer = (*Authorizer)(nil)

// New copies role configuration whose identifiers have already been normalized.
func New(roleSource RoleSource, adminRoleID, mentorRoleID string, overrides map[string]string) *Authorizer {
	return &Authorizer{
		roleSource:          roleSource,
		adminRoleID:         adminRoleID,
		mentorRoleID:        mentorRoleID,
		mentorRoleOverrides: maps.Clone(overrides),
	}
}

func (a *Authorizer) CanCreateClass(actor service_model.Actor) (bool, error) {
	if actor == services.ConfigReconcilerActor() {
		return true, nil
	}

	return a.isAdmin(actor)
}

func (a *Authorizer) CanLinkClassResources(actor service_model.Actor) (bool, error) {
	if actor == services.ConfigReconcilerActor() {
		return true, nil
	}

	return a.isAdmin(actor)
}

func (a *Authorizer) CanClearClass(actor service_model.Actor) (bool, error) {
	return a.isAdmin(actor)
}

func (a *Authorizer) CanDeleteClass(actor service_model.Actor) (bool, error) {
	return a.isAdmin(actor)
}

func (a *Authorizer) CanUnlinkClassResources(actor service_model.Actor) (bool, error) {
	return a.isAdmin(actor)
}

func (a *Authorizer) CanDeleteUser(actor service_model.Actor) (bool, error) {
	return a.isAdmin(actor)
}

func (a *Authorizer) CanExportClass(actor service_model.Actor) (bool, error) {
	return a.isAdmin(actor)
}

func (a *Authorizer) CanManageClasses(actor service_model.Actor, classCodes ...string) (bool, error) {
	if !isUser(actor) || a.roleSource == nil || len(classCodes) == 0 {
		return false, nil
	}

	for _, code := range classCodes {
		if strings.TrimSpace(code) == "" {
			return false, nil
		}
	}

	roles, err := a.roleSource.FetchRoles(actor.ID)
	if err != nil {
		return false, err
	}

	if hasRole(roles, a.adminRoleID) {
		return true, nil
	}

	for _, code := range classCodes {
		role := a.mentorRoleID
		if override, ok := a.mentorRoleOverrides[strings.TrimSpace(code)]; ok {
			role = override
		}

		if !hasRole(roles, role) {
			return false, nil
		}
	}

	return true, nil
}

func (a *Authorizer) isAdmin(actor service_model.Actor) (bool, error) {
	if !isUser(actor) || a.roleSource == nil {
		return false, nil
	}

	roles, err := a.roleSource.FetchRoles(actor.ID)
	if err != nil {
		return false, err
	}

	return hasRole(roles, a.adminRoleID), nil
}

func isUser(actor service_model.Actor) bool {
	return actor.Kind == service_model.ActorUser && strings.TrimSpace(actor.ID) != ""
}

func hasRole(roles []string, role string) bool {
	return role != "" && slices.Contains(roles, role)
}
