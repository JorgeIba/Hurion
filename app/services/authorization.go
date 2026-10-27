package services

import (
	"strings"

	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

func requireManageClasses(auth Authorizer, actor service_model.Actor, classes ...string) error {
	if err := validateAuthorizationInputs(auth, actor); err != nil {
		return err
	}

	if len(classes) == 0 {
		return serviceerr.NewInternalError("could not verify management permissions", nil)
	}

	return requirePermission(auth.CanManageClasses(actor, classes...))
}

// requirePermission translates an authorization decision into service errors.
func requirePermission(allowed bool, err error) error {
	if err != nil {
		return serviceerr.NewInternalError("could not verify permissions", err)
	}

	if !allowed {
		return serviceerr.NewPermissionDeniedError()
	}

	return nil
}

// validateAuthorizationInputs rejects missing dependencies or identities before
// evaluating a permission. Actor kinds are checked by the authorizer itself.
func validateAuthorizationInputs(auth Authorizer, actor service_model.Actor) error {
	if auth == nil || strings.TrimSpace(actor.ID) == "" {
		return serviceerr.NewInternalError("could not verify permissions", nil)
	}

	return nil
}
