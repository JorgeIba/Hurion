package providers

import (
	"fmt"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/providers/mock_providers"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestIsAdmin(t *testing.T) {
	t.Parallel()

	const adminRole = "admin_role"

	tests := []struct {
		name         string
		guildID      string
		userID       string
		injectedRole string
		setupMock    func(ctrl *gomock.Controller) DiscordGoSession
		assertFunc   func(t *testing.T, result bool, err error)
	}{
		{
			name:         "should return true if user has admin role",
			guildID:      "guild_id",
			userID:       "user_id",
			injectedRole: adminRole,
			setupMock: func(ctrl *gomock.Controller) DiscordGoSession {
				mockSession := mock_providers.NewMockDiscordGoSession(ctrl)
				mockSession.EXPECT().GuildMember("guild_id", "user_id").Return(&discordgo.Member{Roles: []string{adminRole}}, nil)
				return mockSession
			},
			assertFunc: func(t *testing.T, result bool, err error) {
				assert.NoError(t, err)
				assert.True(t, result)
			},
		},
		{
			name:         "should return false if user does not have admin role",
			guildID:      "guild_id",
			userID:       "user_id",
			injectedRole: adminRole,
			setupMock: func(ctrl *gomock.Controller) DiscordGoSession {
				mockSession := mock_providers.NewMockDiscordGoSession(ctrl)
				mockSession.EXPECT().GuildMember("guild_id", "user_id").Return(&discordgo.Member{Roles: []string{"other_role"}}, nil)
				return mockSession
			},
			assertFunc: func(t *testing.T, result bool, err error) {
				assert.NoError(t, err)
				assert.False(t, result)
			},
		},
		{
			name:         "should return error if session returns error",
			guildID:      "guild_id",
			userID:       "user_id",
			injectedRole: adminRole,
			setupMock: func(ctrl *gomock.Controller) DiscordGoSession {
				mockSession := mock_providers.NewMockDiscordGoSession(ctrl)
				mockSession.EXPECT().GuildMember("guild_id", "user_id").Return(nil, fmt.Errorf("discord error"))
				return mockSession
			},
			assertFunc: func(t *testing.T, result bool, err error) {
				assert.Error(t, err)
				assert.False(t, result)
			},
		},
		{
			name:         "should return error if session is nil",
			guildID:      "guild_id",
			userID:       "user_id",
			injectedRole: adminRole,
			setupMock: func(ctrl *gomock.Controller) DiscordGoSession {
				return nil
			},
			assertFunc: func(t *testing.T, result bool, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "session not configured")
			},
		},
		{
			name:         "should return error if admin role ID is empty",
			guildID:      "guild_id",
			userID:       "user_id",
			injectedRole: "",
			setupMock: func(ctrl *gomock.Controller) DiscordGoSession {
				mockSession := mock_providers.NewMockDiscordGoSession(ctrl)
				return mockSession
			},
			assertFunc: func(t *testing.T, result bool, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "role ID is not configured")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			provider := NewProviderDiscord(tt.setupMock(ctrl), tt.injectedRole, "mentor_role")
			result, err := provider.IsAdmin(tt.guildID, tt.userID)
			tt.assertFunc(t, result, err)
		})
	}
}

func TestIsMentor(t *testing.T) {
	t.Parallel()

	const mentorRole = "mentor_role"

	tests := []struct {
		name         string
		guildID      string
		userID       string
		injectedRole string
		setupMock    func(ctrl *gomock.Controller) DiscordGoSession
		assertFunc   func(t *testing.T, result bool, err error)
	}{
		{
			name:         "should return true if user has mentor role",
			guildID:      "guild_id",
			userID:       "user_id",
			injectedRole: mentorRole,
			setupMock: func(ctrl *gomock.Controller) DiscordGoSession {
				mockSession := mock_providers.NewMockDiscordGoSession(ctrl)
				mockSession.EXPECT().GuildMember("guild_id", "user_id").Return(&discordgo.Member{Roles: []string{mentorRole}}, nil)
				return mockSession
			},
			assertFunc: func(t *testing.T, result bool, err error) {
				assert.NoError(t, err)
				assert.True(t, result)
			},
		},
		{
			name:         "should return false if user does not have mentor role",
			guildID:      "guild_id",
			userID:       "user_id",
			injectedRole: mentorRole,
			setupMock: func(ctrl *gomock.Controller) DiscordGoSession {
				mockSession := mock_providers.NewMockDiscordGoSession(ctrl)
				mockSession.EXPECT().GuildMember("guild_id", "user_id").Return(&discordgo.Member{Roles: []string{"other_role"}}, nil)
				return mockSession
			},
			assertFunc: func(t *testing.T, result bool, err error) {
				assert.NoError(t, err)
				assert.False(t, result)
			},
		},
		{
			name:         "should return error if session returns error",
			guildID:      "guild_id",
			userID:       "user_id",
			injectedRole: mentorRole,
			setupMock: func(ctrl *gomock.Controller) DiscordGoSession {
				mockSession := mock_providers.NewMockDiscordGoSession(ctrl)
				mockSession.EXPECT().GuildMember("guild_id", "user_id").Return(nil, fmt.Errorf("discord error"))
				return mockSession
			},
			assertFunc: func(t *testing.T, result bool, err error) {
				assert.Error(t, err)
				assert.False(t, result)
			},
		},
		{
			name:         "should return error if mentor role ID is empty",
			guildID:      "guild_id",
			userID:       "user_id",
			injectedRole: "",
			setupMock: func(ctrl *gomock.Controller) DiscordGoSession {
				mockSession := mock_providers.NewMockDiscordGoSession(ctrl)
				return mockSession
			},
			assertFunc: func(t *testing.T, result bool, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "role ID is not configured")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			provider := NewProviderDiscord(tt.setupMock(ctrl), "admin_role", tt.injectedRole)
			result, err := provider.IsMentor(tt.guildID, tt.userID)
			tt.assertFunc(t, result, err)
		})
	}
}
