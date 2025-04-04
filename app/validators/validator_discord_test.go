package validators

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

type MockDiscordSession struct {
	MockGuildMember func(guildID, userID string) (*discordgo.Member, error)
}

func (m *MockDiscordSession) GuildMember(guildID, userID string) (*discordgo.Member, error) {
	return m.MockGuildMember(guildID, userID)
}

func TestHasRole_AdminYes(t *testing.T) {
	const admin_role = "690200603871412245"

	mockSession := &MockDiscordSession{
		MockGuildMember: func(guildID, userID string) (*discordgo.Member, error) {
			return &discordgo.Member{
				Roles: []string{admin_role},
			}, nil
		},
	}

	validator := ValidatorDiscord{session: mockSession}
	result, err := validator.IsAdmin("jorge", "123")
	assert.NoError(t, err)
	assert.True(t, result)
}

func TestHasRole_AdminNo(t *testing.T) {
	const other_role = "other_role"

	mockSession := &MockDiscordSession{
		MockGuildMember: func(guildID, userID string) (*discordgo.Member, error) {
			return &discordgo.Member{
				Roles: []string{other_role},
			}, nil
		},
	}

	validator := ValidatorDiscord{session: mockSession}
	result, err := validator.IsAdmin("jorge", "123")
	assert.NoError(t, err)
	assert.False(t, result)
}

func TestHasRole_MentorYes(t *testing.T) {
	const admin_role = "689627803624407041"

	mockSession := &MockDiscordSession{
		MockGuildMember: func(guildID, userID string) (*discordgo.Member, error) {
			return &discordgo.Member{
				Roles: []string{admin_role},
			}, nil
		},
	}

	validator := ValidatorDiscord{session: mockSession}
	result, err := validator.IsMentor("jorge", "123")
	assert.NoError(t, err)
	assert.True(t, result)
}

func TestHasRole_MentorNo(t *testing.T) {
	const other_role = "other_role"

	mockSession := &MockDiscordSession{
		MockGuildMember: func(guildID, userID string) (*discordgo.Member, error) {
			return &discordgo.Member{
				Roles: []string{other_role},
			}, nil
		},
	}

	validator := ValidatorDiscord{session: mockSession}
	result, err := validator.IsMentor("jorge", "123")
	assert.NoError(t, err)
	assert.False(t, result)
}
