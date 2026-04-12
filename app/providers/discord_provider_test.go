package providers

import (
	"fmt"
	"os"
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
	os.Setenv("DISCORD_ADMIN_ROLE_ID", "admin_role")
	defer os.Unsetenv("DISCORD_ADMIN_ROLE_ID")

	t.Run("should return true if user has admin role", func(t *testing.T) {
		mockSession := &MockDiscordSession{
			MockGuildMember: func(guildID, userID string) (*discordgo.Member, error) {
				return &discordgo.Member{Roles: []string{"admin_role"}}, nil
			},
		}

		provider := NewProviderDiscord(mockSession)
		result, err := provider.IsAdmin("guild_id", "user_id")

		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("should return false if user does not have admin role", func(t *testing.T) {
		mockSession := &MockDiscordSession{
			MockGuildMember: func(guildID, userID string) (*discordgo.Member, error) {
				return &discordgo.Member{Roles: []string{"other_role"}}, nil
			},
		}

		provider := NewProviderDiscord(mockSession)
		result, err := provider.IsAdmin("guild_id", "user_id")

		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("should return error if session returns error", func(t *testing.T) {
		mockSession := &MockDiscordSession{
			MockGuildMember: func(guildID, userID string) (*discordgo.Member, error) {
				return nil, fmt.Errorf("discord error")
			},
		}

		provider := NewProviderDiscord(mockSession)
		result, err := provider.IsAdmin("guild_id", "user_id")

		assert.Error(t, err)
		assert.False(t, result)
	})
}

func TestIsMentor(t *testing.T) {
	os.Setenv("DISCORD_MENTOR_ROLE_ID", "mentor_role")
	defer os.Unsetenv("DISCORD_MENTOR_ROLE_ID")

	t.Run("should return true if user has mentor role", func(t *testing.T) {
		mockSession := &MockDiscordSession{
			MockGuildMember: func(guildID, userID string) (*discordgo.Member, error) {
				return &discordgo.Member{Roles: []string{"mentor_role"}}, nil
			},
		}

		provider := NewProviderDiscord(mockSession)
		result, err := provider.IsMentor("guild_id", "user_id")

		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("should return false if user does not have mentor role", func(t *testing.T) {
		mockSession := &MockDiscordSession{
			MockGuildMember: func(guildID, userID string) (*discordgo.Member, error) {
				return &discordgo.Member{Roles: []string{"other_role"}}, nil
			},
		}

		provider := NewProviderDiscord(mockSession)
		result, err := provider.IsMentor("guild_id", "user_id")

		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("should return error if session returns error", func(t *testing.T) {
		mockSession := &MockDiscordSession{
			MockGuildMember: func(guildID, userID string) (*discordgo.Member, error) {
				return nil, fmt.Errorf("discord error")
			},
		}

		provider := NewProviderDiscord(mockSession)
		result, err := provider.IsMentor("guild_id", "user_id")

		assert.Error(t, err)
		assert.False(t, result)
	})
}
