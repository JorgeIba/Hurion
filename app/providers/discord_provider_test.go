package providers

import (
	"fmt"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

type MockDiscordSession struct {
	MockGuildMember func(guildID, userID string) (*discordgo.Member, error)
}

func (m *MockDiscordSession) GuildMember(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error) {
	return m.MockGuildMember(guildID, userID)
}

func TestIsAdmin(t *testing.T) {
	t.Parallel()

	const adminRole = "admin_role"

	tests := []struct {
		name       string
		guildID    string
		userID     string
		setupMock  func() DiscordGoSession
		assertFunc func(t *testing.T, result bool, err error)
	}{
		{
			name:    "should return true if user has admin role",
			guildID: "guild_id",
			userID:  "user_id",
			setupMock: func() DiscordGoSession {
				return &MockDiscordSession{
					MockGuildMember: func(guildID, userID string) (*discordgo.Member, error) {
						return &discordgo.Member{Roles: []string{adminRole}}, nil
					},
				}
			},
			assertFunc: func(t *testing.T, result bool, err error) {
				assert.NoError(t, err)
				assert.True(t, result)
			},
		},
		{
			name:    "should return false if user does not have admin role",
			guildID: "guild_id",
			userID:  "user_id",
			setupMock: func() DiscordGoSession {
				return &MockDiscordSession{
					MockGuildMember: func(guildID, userID string) (*discordgo.Member, error) {
						return &discordgo.Member{Roles: []string{"other_role"}}, nil
					},
				}
			},
			assertFunc: func(t *testing.T, result bool, err error) {
				assert.NoError(t, err)
				assert.False(t, result)
			},
		},
		{
			name:    "should return error if session returns error",
			guildID: "guild_id",
			userID:  "user_id",
			setupMock: func() DiscordGoSession {
				return &MockDiscordSession{
					MockGuildMember: func(guildID, userID string) (*discordgo.Member, error) {
						return nil, fmt.Errorf("discord error")
					},
				}
			},
			assertFunc: func(t *testing.T, result bool, err error) {
				assert.Error(t, err)
				assert.False(t, result)
			},
		},
		{
			name:    "should return error if session is nil",
			guildID: "guild_id",
			userID:  "user_id",
			setupMock: func() DiscordGoSession {
				return nil
			},
			assertFunc: func(t *testing.T, result bool, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "session not configured")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			provider := NewProviderDiscord(tt.setupMock(), adminRole, "mentor_role")
			result, err := provider.IsAdmin(tt.guildID, tt.userID)
			tt.assertFunc(t, result, err)
		})
	}
}

func TestIsMentor(t *testing.T) {
	t.Parallel()

	const mentorRole = "mentor_role"

	tests := []struct {
		name       string
		guildID    string
		userID     string
		setupMock  func() DiscordGoSession
		assertFunc func(t *testing.T, result bool, err error)
	}{
		{
			name:    "should return true if user has mentor role",
			guildID: "guild_id",
			userID:  "user_id",
			setupMock: func() DiscordGoSession {
				return &MockDiscordSession{
					MockGuildMember: func(guildID, userID string) (*discordgo.Member, error) {
						return &discordgo.Member{Roles: []string{mentorRole}}, nil
					},
				}
			},
			assertFunc: func(t *testing.T, result bool, err error) {
				assert.NoError(t, err)
				assert.True(t, result)
			},
		},
		{
			name:    "should return false if user does not have mentor role",
			guildID: "guild_id",
			userID:  "user_id",
			setupMock: func() DiscordGoSession {
				return &MockDiscordSession{
					MockGuildMember: func(guildID, userID string) (*discordgo.Member, error) {
						return &discordgo.Member{Roles: []string{"other_role"}}, nil
					},
				}
			},
			assertFunc: func(t *testing.T, result bool, err error) {
				assert.NoError(t, err)
				assert.False(t, result)
			},
		},
		{
			name:    "should return error if session returns error",
			guildID: "guild_id",
			userID:  "user_id",
			setupMock: func() DiscordGoSession {
				return &MockDiscordSession{
					MockGuildMember: func(guildID, userID string) (*discordgo.Member, error) {
						return nil, fmt.Errorf("discord error")
					},
				}
			},
			assertFunc: func(t *testing.T, result bool, err error) {
				assert.Error(t, err)
				assert.False(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			provider := NewProviderDiscord(tt.setupMock(), "admin_role", mentorRole)
			result, err := provider.IsMentor(tt.guildID, tt.userID)
			tt.assertFunc(t, result, err)
		})
	}
}
