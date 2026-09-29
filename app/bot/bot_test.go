package bot

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

func TestConfigureSessionIntents(t *testing.T) {
	t.Parallel()

	session := &discordgo.Session{}
	NewBot(nil, nil, nil, BotConfig{}).ConfigureSession(session)

	expected := discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsGuildMessageReactions |
		discordgo.IntentsGuildMembers |
		discordgo.IntentsMessageContent
	assert.Equal(t, expected, session.Identify.Intents)
	assert.True(t, session.StateEnabled)
}

func TestIsWhitelisted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		testMode       bool
		whitelistUsers []string
		whitelistRoles []string
		userID         string
		userRoles      []string
		expected       bool
	}{
		{
			name:           "Test Mode Off - Any User",
			testMode:       false,
			whitelistUsers: []string{"user1"},
			userID:         "user2",
			expected:       true,
		},
		{
			name:           "Test Mode On - Whitelisted User",
			testMode:       true,
			whitelistUsers: []string{"user1", "user2"},
			userID:         "user1",
			expected:       true,
		},
		{
			name:           "Test Mode On - Non-Whitelisted User",
			testMode:       true,
			whitelistUsers: []string{"user1"},
			userID:         "user2",
			expected:       false,
		},
		{
			name:           "Test Mode On - Whitelisted Role",
			testMode:       true,
			whitelistRoles: []string{"role1"},
			userID:         "user2",
			userRoles:      []string{"role1", "role2"},
			expected:       true,
		},
		{
			name:           "Test Mode On - Non-Whitelisted Role",
			testMode:       true,
			whitelistRoles: []string{"role1"},
			userID:         "user2",
			userRoles:      []string{"role3"},
			expected:       false,
		},
		{
			name:           "Test Mode On - Empty Whitelist",
			testMode:       true,
			whitelistUsers: []string{},
			whitelistRoles: []string{},
			userID:         "user1",
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bot := NewBot(nil, nil, nil, BotConfig{
				TestMode:         tt.testMode,
				WhitelistUserIDs: tt.whitelistUsers,
				WhitelistRoleIDs: tt.whitelistRoles,
			})
			assert.Equal(t, tt.expected, bot.isWhitelisted(tt.userID, tt.userRoles))
		})
	}
}
