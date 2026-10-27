package authorization_test

import (
	"errors"
	"testing"

	"github.com/JorgeIba/Hurion/app/authorization"
	"github.com/JorgeIba/Hurion/app/providers/mock_providers"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestDiscordRoleSourceFetchRoles(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name            string
		guildID, userID string
		member          *discordgo.Member
		lookupErr       error
		wantRoles       []string
		wantErr         bool
	}{
		{
			name:    "returns an independent snapshot of current roles",
			guildID: "guild", userID: "actor",
			member:    &discordgo.Member{User: &discordgo.User{ID: "actor"}, Roles: []string{"role-a", "role-b"}},
			wantRoles: []string{"role-a", "role-b"},
		},
		{name: "no roles", guildID: "guild", userID: "actor", member: &discordgo.Member{User: &discordgo.User{ID: "actor"}}},
		{name: "API failure", guildID: "guild", userID: "actor", lookupErr: errors.New("Discord unavailable"), wantErr: true},
		{name: "missing member", guildID: "guild", userID: "actor", wantErr: true},
		{name: "missing member identity", guildID: "guild", userID: "actor", member: &discordgo.Member{Roles: []string{"role-a"}}, wantErr: true},
		{name: "wrong user", guildID: "guild", userID: "actor", member: &discordgo.Member{User: &discordgo.User{ID: "other"}}, wantErr: true},
		{name: "missing server", userID: "actor", wantErr: true},
		{name: "missing user", guildID: "guild", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			members := mock_providers.NewMockDiscordFetcher(gomock.NewController(t))
			if tc.guildID != "" && tc.userID != "" {
				members.EXPECT().FetchMember(tc.guildID, tc.userID).Return(tc.member, tc.lookupErr)
			}
			source := authorization.NewDiscordRoleSource(members, tc.guildID)

			roles, err := source.FetchRoles(tc.userID)

			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, roles)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.wantRoles, roles)
			if len(roles) > 0 {
				roles[0] = "changed"
				assert.Equal(t, tc.wantRoles, tc.member.Roles)
			}
		})
	}
}
