package services

import (
	"errors"

	"github.com/JorgeIba/Hurion/app/authorization"
	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/data"
)

func requireAuthorization(authorizer authorization.Authorizer, actor authorization.Actor, permission authorization.Permission, resource authorization.Resource) error {
	if actor.UserID == "" || actor.GuildID == "" {
		return serviceerr.NewForbiddenError(permission, resource.ClassCode)
	}
	if authorizer == nil {
		return serviceerr.NewAuthorizationUnavailableError(errors.New("authorizer is not configured"))
	}
	err := authorizer.Authorize(actor, permission, resource)
	if errors.Is(err, authorization.ErrDenied) {
		return serviceerr.NewForbiddenError(permission, resource.ClassCode)
	}
	if err != nil {
		return serviceerr.NewAuthorizationUnavailableError(err)
	}
	return nil
}

// RequireAuthorization checks permission for intake UI that has no resolved
// target yet. Mutating services independently enforce target authorization.
func (s *Services) RequireAuthorization(actor authorization.Actor, permission authorization.Permission) error {
	return requireAuthorization(s.authorizer, actor, permission, authorization.Resource{})
}

func authorizeProblem(authorizer authorization.Authorizer, store data.DataStore, actor authorization.Actor, permission authorization.Permission, problemID int64, owner string) error {
	problem, err := store.GetProblemByID(problemID)
	if err != nil {
		if dataerr.IsNotFound(err) {
			return serviceerr.NewProblemNotFoundError("could not resolve the problem", err)
		}
		return serviceerr.MakeGeneralError(err)
	}
	return authorizeProblemClass(authorizer, store, actor, permission, problem.ClassID, owner)
}

func authorizeProblemClass(authorizer authorization.Authorizer, store data.DataStore, actor authorization.Actor, permission authorization.Permission, classID int64, owner string) error {
	class, err := store.GetClassByClassID(classID)
	if err != nil {
		return serviceerr.MakeGeneralError(err)
	}
	return requireAuthorization(authorizer, actor, permission, authorization.Resource{ClassCode: class.ClassCode, OwnerUserID: owner})
}

// CheckAdminAccess authorizes administrative intake before multi-link processing.
func (s *ClassService) CheckAdminAccess(actor authorization.Actor) error {
	return requireAuthorization(s.Authorizer, actor, authorization.Admin, authorization.Resource{})
}

// batchPermissions caches only within one preparation/check. A subsequent
// confirmation creates fresh checks, so role revocations are never cached across requests.
type batchPermissions struct {
	authorizer authorization.Authorizer
	actor      authorization.Actor
	permission authorization.Permission
	decisions  map[string]error
	classes    map[int64]string
}

func newBatchPermissions(authorizer authorization.Authorizer, actor authorization.Actor, permission authorization.Permission) *batchPermissions {
	return &batchPermissions{authorizer: authorizer, actor: actor, permission: permission, decisions: make(map[string]error), classes: make(map[int64]string)}
}
func (b *batchPermissions) class(code string) error {
	if err, found := b.decisions[code]; found {
		return err
	}
	err := requireAuthorization(b.authorizer, b.actor, b.permission, authorization.Resource{ClassCode: code})
	b.decisions[code] = err
	return err
}
func (b *batchPermissions) classID(store data.DataStore, id int64) error {
	code, found := b.classes[id]
	if !found {
		class, err := store.GetClassByClassID(id)
		if err != nil {
			return serviceerr.MakeGeneralError(err)
		}
		code = class.ClassCode
		b.classes[id] = code
	}
	return b.class(code)
}
