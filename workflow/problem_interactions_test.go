package workflow

import (
	"fmt"
	"strings"
	"testing"

	appbot "github.com/JorgeIba/Hurion/app/bot"
	"github.com/JorgeIba/Hurion/app/presentation"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type interactionResponseCall struct {
	interaction *discordgo.Interaction
	response    *discordgo.InteractionResponse
}

type recordingDiscordInteractionClient struct {
	t         *testing.T
	responses []interactionResponseCall
}

func (r *recordingDiscordInteractionClient) InteractionRespond(interaction *discordgo.Interaction, response *discordgo.InteractionResponse, _ ...discordgo.RequestOption) error {
	r.responses = append(r.responses, interactionResponseCall{interaction: interaction, response: response})
	return nil
}

func (r *recordingDiscordInteractionClient) InteractionResponseEdit(_ *discordgo.Interaction, _ *discordgo.WebhookEdit, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	r.t.Helper()
	err := fmt.Errorf("unexpected interaction response edit")
	r.t.Error(err)
	return nil, err
}

func TestWorkflow_ProblemAdd_UsesInjectedResponderForFormAndPreview(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name         string
		channelClass string
		optionClass  string
		wantClass    string
	}{
		{name: "mapped channel supplies its class", channelClass: "basicos_1", wantClass: "basicos_1"},
		{name: "unmapped channel accepts an explicit class", optionClass: "basicos_2", wantClass: "basicos_2"},
		{name: "explicit class overrides the mapped channel", channelClass: "basicos_1", optionClass: "basicos_2", wantClass: "basicos_2"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			env.seedClass(t, "basicos_1", "Basicos 2026")
			env.seedClass(t, "basicos_2", "Basicos 2027")
			if test.channelClass != "" {
				env.seedClassroom(t, testChannelID, test.channelClass)
			}
			var options []*discordgo.ApplicationCommandInteractionDataOption
			if test.optionClass != "" {
				options = append(options, &discordgo.ApplicationCommandInteractionDataOption{Name: "class", Type: discordgo.ApplicationCommandOptionString, Value: test.optionClass})
			}
			command := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				ID: "command_1", Token: "command_token", GuildID: testGuildID, ChannelID: testChannelID,
				Type:   discordgo.InteractionApplicationCommand,
				Member: &discordgo.Member{User: &discordgo.User{ID: "mentor_1"}},
				Data: discordgo.ApplicationCommandInteractionData{
					Name: "problem",
					Options: []*discordgo.ApplicationCommandInteractionDataOption{
						{Name: "add", Type: discordgo.ApplicationCommandOptionSubCommand, Options: options},
					},
				},
			}}

			env.Bot.HandleInteractionEvent(nil, command)

			require.Len(t, env.Responder.responses, 1)
			formCall := env.Responder.responses[0]
			assert.Same(t, command.Interaction, formCall.interaction)
			form := formCall.response
			require.Equal(t, discordgo.InteractionResponseModal, form.Type)
			require.NotNil(t, form.Data)
			require.Len(t, form.Data.Components, 5)

			values := map[string]string{
				"name": "Two Sum 🚀", "url": "https://example.com/two-sum",
				"difficulty": "Medium", "priority": "Recommended", "topic": "Binary search and two pointers",
			}
			data := discordgo.ModalSubmitInteractionData{CustomID: form.Data.CustomID}
			for _, component := range form.Data.Components {
				label := component.(discordgo.Label)
				input := label.Component.(discordgo.TextInput)
				assert.NotEqual(t, "class", input.CustomID, "class is selected before the form opens")
				if input.CustomID == "difficulty" {
					assert.Equal(t, presentation.DifficultyUnspecifiedLabel, input.Value)
				}
				if input.CustomID == "priority" {
					assert.Equal(t, presentation.PriorityUnspecifiedLabel, input.Value)
					require.NotNil(t, input.Required)
					assert.False(t, *input.Required)
				}
				value, ok := values[input.CustomID]
				require.True(t, ok, "unexpected form field %q", input.CustomID)
				data.Components = append(data.Components, &discordgo.Label{
					Component: &discordgo.TextInput{CustomID: input.CustomID, Value: value},
				})
			}
			submit := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				ID: "submit_1", Token: "submit_token", GuildID: testGuildID, ChannelID: testChannelID,
				Type: discordgo.InteractionModalSubmit, Data: data, Member: command.Member,
			}}
			if test.channelClass != "" {
				require.NoError(t, env.Services.Class.RemoveClassroom(testChannelID))
				env.seedClassroom(t, testChannelID, "basicos_2")
			}

			env.Bot.HandleInteractionEvent(nil, submit)

			require.Len(t, env.Responder.responses, 2)
			previewCall := env.Responder.responses[1]
			assert.Same(t, submit.Interaction, previewCall.interaction)
			preview := previewCall.response
			assert.Equal(t, discordgo.InteractionResponseChannelMessageWithSource, preview.Type)
			require.NotNil(t, preview.Data)
			assert.Equal(t, discordgo.MessageFlagsEphemeral, preview.Data.Flags)
			require.NotNil(t, preview.Data.AllowedMentions)
			assert.Empty(t, preview.Data.AllowedMentions.Parse)
			assert.Contains(t, preview.Data.Content, "Two Sum 🚀")
			assert.Contains(t, preview.Data.Content, "Binary search and two pointers")
			assert.Contains(t, preview.Data.Content, `Priority: "Recommended"`)
			assert.Contains(t, preview.Data.Content, fmt.Sprintf("Class: %q", test.wantClass))
			assert.Contains(t, preview.Data.Content, "No problem has been saved.")
			assert.Empty(t, env.Dispatcher.Messages())
			problems, err := env.Services.Problem.GetProblemsGroupedByTopic(test.wantClass)
			require.NoError(t, err)
			assert.Empty(t, problems)
		})
	}
}

func TestWorkflow_ProblemAdd_ReportsClassErrorsBeforeOpeningForm(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, classCode, content string
		seedClass                bool
	}{
		{name: "unmapped channel requires class option", content: "Please use /problem add class:"},
		{name: "unknown explicit class is rejected", classCode: "missing_class", content: "Please check the class code"},
		{name: "class that exceeds the form identifier limit receives a private error", classCode: strings.Repeat("a", 100), seedClass: true, content: "class code cannot fit"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			if test.seedClass {
				env.seedClass(t, test.classCode, "Long class code")
			}
			var options []*discordgo.ApplicationCommandInteractionDataOption
			if test.classCode != "" {
				options = append(options, &discordgo.ApplicationCommandInteractionDataOption{Name: "class", Type: discordgo.ApplicationCommandOptionString, Value: test.classCode})
			}
			event := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				ID: "command_1", Token: "command_token", GuildID: testGuildID, ChannelID: testChannelID,
				Type: discordgo.InteractionApplicationCommand,
				Data: discordgo.ApplicationCommandInteractionData{Name: "problem", Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{Name: "add", Type: discordgo.ApplicationCommandOptionSubCommand, Options: options},
				}},
			}}
			env.Bot.HandleInteractionEvent(nil, event)
			require.Len(t, env.Responder.responses, 1)
			response := env.Responder.responses[0].response
			assert.Equal(t, discordgo.InteractionResponseChannelMessageWithSource, response.Type)
			assert.Equal(t, discordgo.MessageFlagsEphemeral, response.Data.Flags)
			assert.Contains(t, response.Data.Content, test.content)
			assert.Empty(t, env.Dispatcher.Messages())
		})
	}
}

func TestWorkflow_ProblemSubmit_UsesInjectedResponderForPrivateErrors(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name    string
		guildID string
		config  appbot.BotConfig
		content string
	}{
		{
			name: "incomplete submission receives a private error", guildID: testGuildID,
			config: appbot.BotConfig{GuildID: testGuildID}, content: "Could not read the complete form.",
		},
		{
			name: "test mode rejects an account outside the whitelist", guildID: testGuildID,
			config:  appbot.BotConfig{GuildID: testGuildID, TestMode: true, WhitelistUserIDs: []string{"allowed_user"}},
			content: "This development interaction is not enabled for your account.",
		},
		{
			name: "another server receives a private rejection", guildID: "other_guild",
			config: appbot.BotConfig{GuildID: testGuildID}, content: "This interaction is only available in the configured server.",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			env := setupWorkflowEnv(t, test.config)
			event := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				ID: "submit_1", Token: "submit_token", GuildID: test.guildID, ChannelID: testChannelID,
				Type:   discordgo.InteractionModalSubmit,
				Member: &discordgo.Member{User: &discordgo.User{ID: "mentor_1"}},
				Data:   discordgo.ModalSubmitInteractionData{CustomID: "huron:problem:create"},
			}}

			env.Bot.HandleInteractionEvent(nil, event)

			require.Len(t, env.Responder.responses, 1)
			call := env.Responder.responses[0]
			assert.Same(t, event.Interaction, call.interaction)
			assert.Equal(t, discordgo.InteractionResponseChannelMessageWithSource, call.response.Type)
			require.NotNil(t, call.response.Data)
			assert.Equal(t, discordgo.MessageFlagsEphemeral, call.response.Data.Flags)
			assert.Contains(t, call.response.Data.Content, test.content)
			assert.Empty(t, env.Dispatcher.Messages())
		})
	}
}
