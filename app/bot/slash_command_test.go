package bot

import (
	"errors"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
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

func TestUpsertSlashCommands(t *testing.T) {
	api := &fakeApplicationCommandCreator{}
	require.NoError(t, upsertSlashCommands(api, " app ", "guild"))
	require.Len(t, api.commands, 2)
	assert.Equal(t, "problem", api.commands[0].Name)
	assert.Equal(t, "ac", api.commands[1].Name)
	assert.Equal(t, "app", api.applicationID)
	assert.Equal(t, "guild", api.guildID)
}
func TestUpsertSlashCommandsStopsAtFailure(t *testing.T) {
	offline := errors.New("offline")
	api := &fakeApplicationCommandCreator{err: offline}
	err := upsertSlashCommands(api, "app", "guild")
	require.ErrorIs(t, err, offline)
	require.Len(t, api.commands, 1)
	assert.ErrorContains(t, err, "/problem")
	assert.Error(t, UpsertSlashCommands(nil, "app", "guild"))
}
