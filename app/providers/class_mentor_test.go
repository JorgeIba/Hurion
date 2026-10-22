package providers_test

import (
	"errors"
	"github.com/JorgeIba/Hurion/app/providers/mock_providers"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
	"testing"

	"github.com/JorgeIba/Hurion/app/providers"
	"github.com/JorgeIba/Hurion/app/script"
	"github.com/stretchr/testify/assert"
)

func TestClassMentorPermissions(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, role, class string
		allowed           bool
	}{
		{"inherits global", "mentor", "basics", true},
		{"global replaced", "mentor", "advanced", false},
		{"custom mentor", "advanced-mentor", "advanced", true},
		{"custom cannot manage other classes", "advanced-mentor", "basics", false},
		{"empty override denies global", "mentor", "disabled", false},
		{"admin preserved", "admin", "advanced", true},
		{"admin preserved with empty list", "admin", "disabled", true},
		{"student denied", "student", "advanced", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			raw := script.NewFakeDiscordProvider("admin", "mentor")
			raw.SetUserRoles("actor", []string{tt.role})
			scoped := providers.WithClassMentorRoles(raw, map[string][]string{"advanced": {"advanced-mentor"}, "disabled": {}})
			allowed, err := scoped.(providers.ClassMentorValidator).CanManageClass("guild", "actor", tt.class)
			assert.NoError(t, err)
			assert.Equal(t, tt.allowed, allowed)
		})
	}
}

func TestClassMentorPolicyCopiesConfigurationAndAllowsIntake(t *testing.T) {
	t.Parallel()
	raw := script.NewFakeDiscordProvider("admin", "global")
	raw.SetUserRoles("actor", []string{"custom"})
	roles := map[string][]string{"advanced": {"custom", "second-custom"}}
	scoped := providers.WithClassMentorRoles(raw, roles)
	roles["advanced"][0] = "changed"
	delete(roles, "advanced")
	allowed, err := scoped.IsMentor("guild", "actor")
	assert.NoError(t, err)
	assert.True(t, allowed, "custom mentors must be able to open intake forms")
	allowed, err = scoped.(providers.ClassMentorValidator).CanManageClass("guild", "actor", " advanced ")
	assert.NoError(t, err)
	assert.True(t, allowed, "policy must be immutable and normalize class whitespace like problem writes")
}

func TestClassMentorLookupFailuresDenyAccess(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name                string
		adminErr, memberErr bool
	}{
		{name: "admin role lookup fails", adminErr: true},
		{name: "custom role lookup fails", memberErr: true},
		{name: "member response is missing"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			session := mock_providers.NewMockDiscordGoSession(gomock.NewController(t))
			raw := providers.NewProviderDiscord(session, "admin", "global")
			failure := errors.New("Discord unavailable")
			var adminErr error
			if tt.adminErr {
				adminErr = failure
			}
			session.EXPECT().GuildMember("guild", "actor").Return(&discordgo.Member{}, adminErr)
			if !tt.adminErr {
				var memberErr error
				if tt.memberErr {
					memberErr = failure
				}
				session.EXPECT().GuildMember("guild", "actor").Return(nil, memberErr)
			}
			scoped := providers.WithClassMentorRoles(raw, map[string][]string{"advanced": {"custom"}})
			allowed, err := scoped.(providers.ClassMentorValidator).CanManageClass("guild", "actor", "advanced")
			assert.False(t, allowed)
			assert.Error(t, err)
		})
	}
}
