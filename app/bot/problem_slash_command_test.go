package bot

import (
	"errors"
	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpsertProblemSlashCommandCreatesGuildScopedProblemCommand(t *testing.T) {
	api := &fakeApplicationCommandCreator{}

	err := upsertSlashCommand(api, "application-123", "guild-456", discordcommands.ProblemSlashCommand(service_model.ClassCodeMaxLength))

	require.NoError(t, err)
	require.Len(t, api.commands, 1)
	assert.Equal(t, "application-123", api.applicationID)
	assert.Equal(t, "guild-456", api.guildID)
	assert.Equal(t, "problem", api.commands[0].Name)
	assert.Equal(t, discordgo.ChatApplicationCommand, api.commands[0].Type)
	require.Len(t, api.commands[0].Options, 4)
	assert.Equal(t, "add", api.commands[0].Options[0].Name)
	assert.Equal(t, discordgo.ApplicationCommandOptionSubCommand, api.commands[0].Options[0].Type)
	require.Len(t, api.commands[0].Options[0].Options, 1)
	class := api.commands[0].Options[0].Options[0]
	assert.Equal(t, "class", class.Name)
	assert.Equal(t, discordgo.ApplicationCommandOptionString, class.Type)
	assert.False(t, class.Required, "the channel can supply the class")
	assert.Zero(t, class.MaxLength, "class length is validated by the service")
	edit := api.commands[0].Options[1]
	assert.Equal(t, "edit", edit.Name)
	require.Len(t, edit.Options, 2)
	assert.Equal(t, "problem", edit.Options[0].Name)
	assert.True(t, edit.Options[0].Required)
	assert.False(t, edit.Options[0].Autocomplete)
	assert.Equal(t, "class", edit.Options[1].Name)
	assert.False(t, edit.Options[1].Required)

	assert.Equal(t, "add_batch", api.commands[0].Options[2].Name)
	assert.Equal(t, discordgo.ApplicationCommandOptionSubCommand, api.commands[0].Options[2].Type)
	assert.Equal(t, "edit_batch", api.commands[0].Options[3].Name)
	assert.Equal(t, discordgo.ApplicationCommandOptionSubCommand, api.commands[0].Options[3].Type)
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

			err := upsertSlashCommand(api, test.applicationID, test.guildID, discordcommands.ProblemSlashCommand(service_model.ClassCodeMaxLength))

			require.Error(t, err)
			assert.Empty(t, api.commands)
		})
	}
}

func TestUpsertProblemSlashCommandReturnsDiscordError(t *testing.T) {
	api := &fakeApplicationCommandCreator{err: errors.New("Discord API unavailable")}

	err := upsertSlashCommand(api, "application-123", "guild-456", discordcommands.ProblemSlashCommand(service_model.ClassCodeMaxLength))

	require.ErrorContains(t, err, "upsert guild slash command /problem")
	assert.ErrorContains(t, err, "Discord API unavailable")
}
