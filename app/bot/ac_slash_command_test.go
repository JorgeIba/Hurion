package bot

import (
	"errors"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

func TestUpsertACSlashCommand(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, app, guild string
		apiErr           error
		wantError        string
	}{
		{name: "register AC batch", app: "app", guild: "guild"},
		{name: "missing application", guild: "guild", wantError: "application ID"},
		{name: "missing guild", app: "app", wantError: "guild ID"},
		{name: "Discord failure", app: "app", guild: "guild", apiErr: errors.New("offline"), wantError: "/ac: offline"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			api := &fakeApplicationCommandCreator{err: test.apiErr}
			err := upsertACSlashCommand(api, test.app, test.guild)
			if test.wantError != "" {
				assert.ErrorContains(t, err, test.wantError)
				return
			}
			assert.NoError(t, err)
			if !assert.Len(t, api.commands, 1) {
				return
			}
			command := api.commands[0]
			assert.Equal(t, "ac", command.Name)
			if assert.Len(t, command.Options, 1) {
				assert.Equal(t, "add_batch", command.Options[0].Name)
				assert.Equal(t, discordgo.ApplicationCommandOptionSubCommand, command.Options[0].Type)
				assert.Empty(t, command.Options[0].Options)
			}
		})
	}
}
