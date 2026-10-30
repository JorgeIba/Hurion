package bot

import (
	"testing"

	"github.com/JorgeIba/Hurion/app/context"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type autocompleteResponder struct {
	dispatcher.InteractionResponder
	responses       []*discordgo.InteractionResponse
	privateMessages []string
}

func (r *autocompleteResponder) Respond(_ *discordgo.Interaction, response *discordgo.InteractionResponse) error {
	r.responses = append(r.responses, response)
	return nil
}

func (r *autocompleteResponder) RespondPrivately(_ *discordgo.Interaction, message string) error {
	r.privateMessages = append(r.privateMessages, message)
	return nil
}

type autocompleteCommandProbe struct {
	context.ProblemInteractionController
	context.ACInteractionController
	calls int
}

func (p *autocompleteCommandProbe) HandleProposeACInteraction(*discordgo.InteractionCreate) {
	p.calls++
}

func (p *autocompleteCommandProbe) HandleEditProblemInteraction(*discordgo.InteractionCreate) {
	p.calls++
}

func TestAutocompleteRestrictionsReturnEmptySuggestionsWithoutReadingCatalog(t *testing.T) {
	t.Parallel()
	for _, route := range []struct {
		command    string
		subcommand string
		option     string
	}{
		{command: "ac", subcommand: "submit", option: "id"},
		{command: "ac", subcommand: "remove", option: "id"},
		{command: "problem", subcommand: "edit", option: "problem"},
		{command: "problem", subcommand: "remove", option: "problem"},
	} {
		for _, test := range []struct {
			name     string
			guildID  string
			testMode bool
		}{
			{name: "wrong server", guildID: "another-server"},
			{name: "excluded from test mode", guildID: "guild", testMode: true},
		} {
			t.Run(route.command+"/"+test.name, func(t *testing.T) {
				t.Parallel()
				// No datastore calls are expected: even a read is a failure here.
				store := mock_data.NewMockDataStore(gomock.NewController(t))
				responder := &autocompleteResponder{}
				commands := &autocompleteCommandProbe{}
				bot := NewBot(services.NewServices(store, nil), nil, nil, responder, BotConfig{
					GuildID:          "guild",
					TestMode:         test.testMode,
					WhitelistUserIDs: []string{"another-user"},
				})
				event := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
					ID:        "autocomplete",
					Type:      discordgo.InteractionApplicationCommandAutocomplete,
					GuildID:   test.guildID,
					ChannelID: "channel",
					Member: &discordgo.Member{
						User:  &discordgo.User{ID: "student"},
						Roles: []string{"unlisted-role"},
					},
					Data: discordgo.ApplicationCommandInteractionData{
						Name: route.command,
						Options: []*discordgo.ApplicationCommandInteractionDataOption{{
							Name: route.subcommand,
							Type: discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandInteractionDataOption{{
								Name:    route.option,
								Type:    discordgo.ApplicationCommandOptionString,
								Value:   "sum",
								Focused: true,
							}},
						}},
					},
				}}
				bot.HandleInteractionEvent(nil, event, commands, commands)
				require.Len(t, responder.responses, 1)
				response := responder.responses[0]
				assert.Equal(t, discordgo.InteractionApplicationCommandAutocompleteResult, response.Type)
				require.NotNil(t, response.Data)
				assert.NotNil(t, response.Data.Choices)
				assert.Empty(t, response.Data.Choices)
				assert.Empty(t, response.Data.Content)
				assert.Empty(t, responder.privateMessages)
				assert.Zero(t, commands.calls)
			})
		}
	}
}
