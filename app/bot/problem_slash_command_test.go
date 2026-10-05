package bot

import (
	"errors"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeApplicationCommandCreator struct {
	applicationID string
	guildID       string
	commands      []*discordgo.ApplicationCommand
	err           error
}

func (fake *fakeApplicationCommandCreator) ApplicationCommandCreate(appID, guildID string, command *discordgo.ApplicationCommand, _ ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
	fake.applicationID = appID
	fake.guildID = guildID
	fake.commands = append(fake.commands, command)
	return command, fake.err
}

func TestUpsertProblemSlashCommandCreatesGuildScopedProblemCommand(t *testing.T) {
	api := &fakeApplicationCommandCreator{}

	err := upsertProblemSlashCommand(api, "application-123", "guild-456")

	require.NoError(t, err)
	require.Len(t, api.commands, 1)
	assert.Equal(t, "application-123", api.applicationID)
	assert.Equal(t, "guild-456", api.guildID)
	assert.Equal(t, "problem", api.commands[0].Name)
	assert.Equal(t, discordgo.ChatApplicationCommand, api.commands[0].Type)
	require.Len(t, api.commands[0].Options, 1)
	assert.Equal(t, "add", api.commands[0].Options[0].Name)
	assert.Equal(t, discordgo.ApplicationCommandOptionSubCommand, api.commands[0].Options[0].Type)
	require.Len(t, api.commands[0].Options[0].Options, 1)
	class := api.commands[0].Options[0].Options[0]
	assert.Equal(t, "class", class.Name)
	assert.Equal(t, discordgo.ApplicationCommandOptionString, class.Type)
	assert.False(t, class.Required, "the channel can supply the class")
}

func TestUpsertProblemSlashCommandRejectsMissingIDsBeforeCallingDiscord(t *testing.T) {
	tests := []struct {
		name          string
		applicationID string
		guildID       string
	}{
		{name: "missing application ID", guildID: "guild-456"},
		{name: "missing guild ID", applicationID: "application-123"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api := &fakeApplicationCommandCreator{}

			err := upsertProblemSlashCommand(api, test.applicationID, test.guildID)

			require.Error(t, err)
			assert.Empty(t, api.commands)
		})
	}
}

func TestUpsertProblemSlashCommandReturnsDiscordError(t *testing.T) {
	api := &fakeApplicationCommandCreator{err: errors.New("Discord API unavailable")}

	err := upsertProblemSlashCommand(api, "application-123", "guild-456")

	require.ErrorContains(t, err, "upsert guild slash command /problem")
	assert.ErrorContains(t, err, "Discord API unavailable")
}
