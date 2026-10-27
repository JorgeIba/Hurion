package authorization_test

import (
	"errors"
	"testing"

	"github.com/JorgeIba/Hurion/app/authorization"
	"github.com/JorgeIba/Hurion/app/authorization/mock_authorization"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestAuthorizerClassPermissions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name           string
		roles, classes []string
		allowed        bool
	}{
		{"inherits global", []string{"global"}, []string{"basics"}, true},
		{"override replaces global", []string{"global"}, []string{"advanced"}, false},
		{"class mentor needs no global role", []string{"advanced-mentor"}, []string{"advanced"}, true},
		{"class mentor cannot manage another class", []string{"advanced-mentor"}, []string{"basics"}, false},
		{"all classes required", []string{"global"}, []string{"basics", "advanced"}, false},
		{"both roles", []string{"global", "advanced-mentor"}, []string{"basics", "advanced"}, true},
		{"administrator", []string{"admin"}, []string{"basics", "advanced"}, true},
		{"student", []string{"student"}, []string{"basics"}, false},
		{"whitespace normalized", []string{"advanced-mentor"}, []string{" advanced "}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			source := mock_authorization.NewMockRoleSource(gomock.NewController(t))
			// One lookup covers all classes, including the admin check.
			source.EXPECT().FetchRoles("actor").Return(tc.roles, nil)
			auth := authorization.New(source, "admin", "global", map[string]string{"advanced": "advanced-mentor"})

			allowed, err := auth.CanManageClasses(services.UserActor("actor"), tc.classes...)

			assert.NoError(t, err)
			assert.Equal(t, tc.allowed, allowed)
		})
	}
}

func TestAuthorizerCopiesConfigAndRechecksMembership(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                      string
		firstRoles, nextRoles     []string
		firstAllowed, nextAllowed bool
	}{
		{name: "revoked role", firstRoles: []string{"specific"}, firstAllowed: true},
		{name: "granted role", nextRoles: []string{"specific"}, nextAllowed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			source := mock_authorization.NewMockRoleSource(gomock.NewController(t))
			overrides := map[string]string{"advanced": "specific"}
			auth := authorization.New(source, "admin", "global", overrides)
			overrides["advanced"] = "changed"
			gomock.InOrder(
				source.EXPECT().FetchRoles("actor").Return(tc.firstRoles, nil),
				source.EXPECT().FetchRoles("actor").Return(tc.nextRoles, nil),
			)

			allowed, err := auth.CanManageClasses(services.UserActor("actor"), "advanced")
			assert.NoError(t, err)
			assert.Equal(t, tc.firstAllowed, allowed)
			allowed, err = auth.CanManageClasses(services.UserActor("actor"), "advanced")
			assert.NoError(t, err)
			assert.Equal(t, tc.nextAllowed, allowed)
		})
	}
}

func TestAuthorizerActorPermissions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                                              string
		actor                                             service_model.Actor
		roles                                             []string
		wantCreate, wantLink, wantDestructive, wantManage bool
		sourceMissing                                     bool
	}{
		{name: "administrator", actor: services.UserActor("actor"), roles: []string{"admin"}, wantCreate: true, wantLink: true, wantDestructive: true, wantManage: true},
		{name: "mentor", actor: services.UserActor("actor"), roles: []string{"global"}, wantManage: true},
		{name: "student", actor: services.UserActor("actor"), roles: []string{"student"}},
		{name: "configuration reconciler", actor: services.ConfigReconcilerActor(), wantCreate: true, wantLink: true},
		{name: "unknown process", actor: service_model.Actor{ID: "unknown", Kind: service_model.ActorProcess}},
		{name: "process with admin user ID", actor: service_model.Actor{ID: "actor", Kind: service_model.ActorProcess}},
		{name: "missing identity"},
		{name: "empty user", actor: services.UserActor("")},
		{name: "role source unavailable", actor: services.UserActor("actor"), sourceMissing: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			source := mock_authorization.NewMockRoleSource(gomock.NewController(t))
			var roleSource authorization.RoleSource = source
			if tc.sourceMissing {
				roleSource = nil
			}
			auth := authorization.New(roleSource, "admin", "global", nil)
			operations := []struct {
				name    string
				check   func(service_model.Actor) (bool, error)
				allowed bool
			}{
				{"create class", auth.CanCreateClass, tc.wantCreate},
				{"link resources", auth.CanLinkClassResources, tc.wantLink},
				{"clear class", auth.CanClearClass, tc.wantDestructive},
				{"delete class", auth.CanDeleteClass, tc.wantDestructive},
				{"unlink resources", auth.CanUnlinkClassResources, tc.wantDestructive},
				{"delete user", auth.CanDeleteUser, tc.wantDestructive},
				{"export class", auth.CanExportClass, tc.wantDestructive},
				{"manage classes", func(actor service_model.Actor) (bool, error) { return auth.CanManageClasses(actor, "basics") }, tc.wantManage},
			}
			if !tc.sourceMissing && tc.actor.Kind == service_model.ActorUser && tc.actor.ID != "" {
				source.EXPECT().FetchRoles(tc.actor.ID).Return(tc.roles, nil).Times(len(operations))
			}

			for _, operation := range operations {
				allowed, err := operation.check(tc.actor)
				assert.NoError(t, err, operation.name)
				assert.Equal(t, operation.allowed, allowed, operation.name)
			}
		})
	}
}

func TestAuthorizerRejectsInvalidRequestsBeforeRoleLookup(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		actor   service_model.Actor
		classes []string
	}{
		{name: "no classes", actor: services.UserActor("actor")},
		{name: "empty class", actor: services.UserActor("actor"), classes: []string{""}},
		{name: "blank class in batch", actor: services.UserActor("actor"), classes: []string{"basics", " "}},
		{name: "missing user", actor: services.UserActor(""), classes: []string{"basics"}},
		{name: "process", actor: services.ConfigReconcilerActor(), classes: []string{"basics"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			source := mock_authorization.NewMockRoleSource(gomock.NewController(t))
			auth := authorization.New(source, "admin", "global", nil)
			allowed, err := auth.CanManageClasses(tc.actor, tc.classes...)
			assert.NoError(t, err)
			assert.False(t, allowed)
		})
	}
}

func TestAuthorizerRejectsRoleLookupFailures(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		check func(*authorization.Authorizer, service_model.Actor) (bool, error)
	}{
		{"admin operation", (*authorization.Authorizer).CanCreateClass},
		{"class operation", func(a *authorization.Authorizer, actor service_model.Actor) (bool, error) {
			return a.CanManageClasses(actor, "basics")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			source := mock_authorization.NewMockRoleSource(gomock.NewController(t))
			lookupErr := errors.New("role lookup unavailable")
			source.EXPECT().FetchRoles("actor").Return([]string{"admin"}, lookupErr)
			auth := authorization.New(source, "admin", "global", nil)
			allowed, err := tc.check(auth, services.UserActor("actor"))
			assert.False(t, allowed)
			assert.ErrorIs(t, err, lookupErr)
		})
	}
}
