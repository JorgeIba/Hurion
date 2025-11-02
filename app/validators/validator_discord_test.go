package validators

import (
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

func TestHasRole_AdminYes(t *testing.T) {
	const admin_role = "test_admin_role_id"
	
	// Set environment variable for test
	os.Setenv("DISCORD_ADMIN_ROLE_ID", admin_role)
	defer os.Unsetenv("DISCORD_ADMIN_ROLE_ID")

	mockSession := &MockDiscordSession{
		MockGuildMember: func(guildID, userID string) (*discordgo.Member, error) {
			return &discordgo.Member{
				Roles: []string{admin_role},
			}, nil
		},
	}

	validator := NewValidatorDiscord(mockSession)
	result, err := validator.IsAdmin("jorge", "123")
	assert.NoError(t, err)
	assert.True(t, result)
}

func TestHasRole_AdminNo(t *testing.T) {
	const admin_role = "test_admin_role_id"
	const other_role = "other_role"
	
	// Set environment variable for test
	os.Setenv("DISCORD_ADMIN_ROLE_ID", admin_role)
	defer os.Unsetenv("DISCORD_ADMIN_ROLE_ID")

	mockSession := &MockDiscordSession{
		MockGuildMember: func(guildID, userID string) (*discordgo.Member, error) {
			return &discordgo.Member{
				Roles: []string{other_role},
			}, nil
		},
	}

	validator := NewValidatorDiscord(mockSession)
	result, err := validator.IsAdmin("jorge", "123")
	assert.NoError(t, err)
	assert.False(t, result)
}

func TestHasRole_MentorYes(t *testing.T) {
	const mentor_role = "test_mentor_role_id"
	
	// Set environment variable for test
	os.Setenv("DISCORD_MENTOR_ROLE_ID", mentor_role)
	defer os.Unsetenv("DISCORD_MENTOR_ROLE_ID")

	mockSession := &MockDiscordSession{
		MockGuildMember: func(guildID, userID string) (*discordgo.Member, error) {
			return &discordgo.Member{
				Roles: []string{mentor_role},
			}, nil
		},
	}

	validator := NewValidatorDiscord(mockSession)
	result, err := validator.IsMentor("jorge", "123")
	assert.NoError(t, err)
	assert.True(t, result)
}

func TestHasRole_MentorNo(t *testing.T) {
	const mentor_role = "test_mentor_role_id"
	const other_role = "other_role"
	
	// Set environment variable for test
	os.Setenv("DISCORD_MENTOR_ROLE_ID", mentor_role)
	defer os.Unsetenv("DISCORD_MENTOR_ROLE_ID")

	mockSession := &MockDiscordSession{
		MockGuildMember: func(guildID, userID string) (*discordgo.Member, error) {
			return &discordgo.Member{
				Roles: []string{other_role},
			}, nil
		},
	}

	validator := NewValidatorDiscord(mockSession)
	result, err := validator.IsMentor("jorge", "123")
	assert.NoError(t, err)
	assert.False(t, result)
}
