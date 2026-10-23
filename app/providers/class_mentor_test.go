package providers_test

import (
	"errors"
	"testing"

	"github.com/JorgeIba/Hurion/app/authorization"
	"github.com/JorgeIba/Hurion/app/providers"
	"github.com/JorgeIba/Hurion/app/providers/mock_providers"
	"github.com/JorgeIba/Hurion/app/script"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestDiscordAuthorizer(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, role, actor, owner, class string
		permission                      authorization.Permission
		allowed                         bool
	}{
		{"inherits global", "mentor", "actor", "student", "basics", authorization.ManageACs, true},
		{"global role replaced", "mentor", "actor", "student", "advanced", authorization.ManageACs, false},
		{"custom role", "custom", "actor", "student", "advanced", authorization.ManageACs, true},
		{"second custom role", "custom2", "actor", "student", "advanced", authorization.ManageProblems, true},
		{"custom is not global", "custom", "actor", "student", "basics", authorization.ManageProblems, false},
		{"empty replaces global", "mentor", "actor", "student", "disabled", authorization.ManageProblems, false},
		{"admin preserved", "admin", "actor", "student", "advanced", authorization.ManageACs, true},
		{"admin with empty override", "admin", "actor", "student", "disabled", authorization.ManageProblems, true},
		{"self undo", "student", "student", "student", "advanced", authorization.UndoAC, true},
		{"other undo denied", "student", "actor", "student", "advanced", authorization.UndoAC, false},
		{"custom undo", "custom", "actor", "student", "advanced", authorization.UndoAC, true},
		{"admin-only denied to mentor", "custom", "actor", "student", "advanced", authorization.Admin, false},
		{"custom intake", "custom", "actor", "student", "", authorization.MentorIntake, true},
		{"student access", "student", "actor", "student", "advanced", authorization.AccessClass, true},
		{"mentor does not imply student access", "custom", "actor", "student", "advanced", authorization.AccessClass, false},
		{"unknown denied to admin", "admin", "actor", "student", "advanced", "unknown", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			raw := script.NewFakeDiscordProvider()
			raw.SetUserRoles(tt.actor, []string{tt.role})
			authorizer := providers.NewDiscordAuthorizer(raw, "admin", "mentor", map[string][]string{"advanced": {"custom", "custom2"}, "disabled": {}})
			err := authorizer.Authorize(authorization.Actor{GuildID: "guild", UserID: tt.actor}, tt.permission, authorization.Resource{ClassCode: tt.class, OwnerUserID: tt.owner, AccessRoleIDs: []string{"student"}})
			if tt.allowed {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, authorization.ErrDenied)
			}
		})
	}
}

func TestDiscordAuthorizerRequiresFreshMembership(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name      string
		missing   bool
		lookupErr error
	}{
		{name: "missing member", missing: true},
		{name: "lookup fails", lookupErr: errors.New("Discord unavailable")},
		{name: "valid member"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fetcher := mock_providers.NewMockDiscordFetcher(gomock.NewController(t))
			var member *discordgo.Member
			if !tt.missing {
				member = &discordgo.Member{Roles: []string{"custom"}}
			}
			fetcher.EXPECT().FetchMember("guild", "actor").Return(member, tt.lookupErr)
			a := providers.NewDiscordAuthorizer(fetcher, "admin", "mentor", map[string][]string{"advanced": {"custom"}})
			err := a.Authorize(authorization.Actor{GuildID: "guild", UserID: "actor"}, authorization.ManageACs, authorization.Resource{ClassCode: "advanced"})
			if tt.missing || tt.lookupErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDiscordAuthorizerSelfUndoDoesNotNeedDiscord(t *testing.T) {
	t.Parallel()
	a := providers.NewDiscordAuthorizer(nil, "admin", "mentor", nil)
	assert.NoError(t, a.Authorize(authorization.Actor{GuildID: "guild", UserID: "student"}, authorization.UndoAC, authorization.Resource{OwnerUserID: "student"}))
	assert.ErrorIs(t, a.Authorize(authorization.Actor{}, authorization.UndoAC, authorization.Resource{}), authorization.ErrDenied)
}
