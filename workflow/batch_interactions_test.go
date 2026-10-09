package workflow

import (
	"errors"
	"testing"

	appbot "github.com/JorgeIba/Hurion/app/bot"
	"github.com/JorgeIba/Hurion/app/controllers/mentor"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

func TestWorkflow_BatchCommandsOpenFormsAndParseWithoutWrites(t *testing.T) {
	t.Parallel()
	for _, test := range []struct{ name, command, input, want string }{
		{name: "problem JSON", command: "problem", input: `[{"class":"basicos","name":"Two Sum","url":"https://example.com"}]`, want: "Entries: 1"},
		{name: "AC JSON", command: "ac", input: `[{"user_id":"123","problem_id":42}]`, want: "Entries: 1"},
		{name: "repeated ACs preserved", command: "ac", input: `[{"user_id":"123","problem_id":42},{"user_id":"123","problem_id":42}]`, want: "Entries: 2"},
		{name: "syntax error", command: "problem", input: `[{`, want: "line 1, column"},
		{name: "unknown fields", command: "problem", input: `[{"unknown":true}]`, want: "Entry 1"},
		{name: "duplicate fields", command: "ac", input: `[{"user_id":"123","user_id":"456"}]`, want: "Entry 1"},
		{name: "wrong type", command: "ac", input: `[{"problem_id":"42"}]`, want: "Entry 1"},
		{name: "multiple invalid entries", command: "ac", input: `[null,{"problem_id":"42"}]`, want: "Entry 2"},
		{name: "empty source", command: "ac", input: "", want: "paste JSON"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			classID := env.seedClass(t, "basicos", "Basicos")
			checks := 0
			env.Discord.isMentorFunc = func(_, _ string) (bool, error) { checks++; return true, nil }
			member := &discordgo.Member{User: &discordgo.User{ID: "mentor"}}
			env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand, GuildID: testGuildID, Member: member,
				Data: discordgo.ApplicationCommandInteractionData{Name: test.command, Options: []*discordgo.ApplicationCommandInteractionDataOption{{Name: "add_batch", Type: discordgo.ApplicationCommandOptionSubCommand}}},
			}})
			if !assert.Len(t, env.Responder.responses, 1) {
				return
			}
			response := env.Responder.responses[0].response
			assert.Equal(t, discordgo.InteractionResponseModal, response.Type)
			expectedRoute := discord.ACBatchModalCustomID
			if test.command == "problem" {
				expectedRoute = discord.ProblemBatchModalCustomID
			}
			assert.Equal(t, expectedRoute, response.Data.CustomID)
			env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionModalSubmit, GuildID: testGuildID, Member: member,
				Data: discordgo.ModalSubmitInteractionData{CustomID: response.Data.CustomID, Components: []discordgo.MessageComponent{
					&discordgo.Label{Component: &discordgo.TextInput{CustomID: "json", Value: test.input}},
					&discordgo.Label{Component: &discordgo.FileUpload{CustomID: "file"}},
				}},
			}})
			if assert.Len(t, env.Responder.responses, 2) {
				assert.Equal(t, discordgo.InteractionResponseDeferredChannelMessageWithSource, env.Responder.responses[1].response.Type)
				assert.Equal(t, discordgo.MessageFlagsEphemeral, env.Responder.responses[1].response.Data.Flags)
			}
			if assert.Len(t, env.Responder.edits, 1) {
				assert.Contains(t, *env.Responder.edits[0].response.Content, test.want)
				assert.Empty(t, env.Responder.edits[0].response.AllowedMentions.Parse)
			}
			assert.Equal(t, 2, checks, "authorize at open and submit")
			problems, err := env.DB.GetProblemsByClass(classID)
			assert.NoError(t, err)
			assert.Empty(t, problems, "intake must not create problems")
			_, err = env.DB.GetUserByUserID("123")
			assert.Error(t, err, "intake must not create student records")
			solved, err := env.DB.HasUserSolvedProblem("123", 42)
			assert.NoError(t, err)
			assert.False(t, solved, "intake must not register ACs")
		})
	}
}

func TestWorkflow_BatchInteractionsHonorAuthorityAndBotGates(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, command, guild                                     string
		admin, denied, missingMember, testMode, submit, ackFails bool
		want                                                     string
	}{
		{name: "mentor denied", command: "problem", denied: true, want: "mentor or admin"},
		{name: "AC denied", command: "ac", denied: true, want: "mentor or admin"},
		{name: "missing member", command: "ac", missingMember: true, want: "identified server member"},
		{name: "wrong server", command: "ac", guild: "other", want: "configured server"},
		{name: "test mode whitelist", command: "ac", testMode: true, want: "not enabled"},
		{name: "admin opens AC form", command: "ac", admin: true},
		{name: "admin opens problem form", command: "problem", admin: true},
		{name: "authority revoked before submit", command: "ac", denied: true, submit: true, want: "mentor or admin"},
		{name: "failed ack stops processing", command: "ac", submit: true, ackFails: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID, TestMode: test.testMode})
			checks := 0
			env.Discord.isMentorFunc = func(_, _ string) (bool, error) { checks++; return !test.denied && !test.admin, nil }
			env.Discord.isAdminFunc = func(_, _ string) (bool, error) { return test.admin, nil }
			guild := test.guild
			if guild == "" {
				guild = testGuildID
			}
			member := &discordgo.Member{User: &discordgo.User{ID: "mentor"}, Roles: []string{"role"}}
			if test.missingMember {
				member = nil
			}
			var data discordgo.InteractionData = discordgo.ApplicationCommandInteractionData{Name: test.command, Options: []*discordgo.ApplicationCommandInteractionDataOption{{Name: "add_batch", Type: discordgo.ApplicationCommandOptionSubCommand}}}
			kind := discordgo.InteractionApplicationCommand
			if test.submit {
				kind = discordgo.InteractionModalSubmit
				data = discordgo.ModalSubmitInteractionData{CustomID: discord.ACBatchModalCustomID}
			}
			if test.ackFails {
				env.Responder.respondFunc = func(*discordgo.InteractionResponse) error { return errors.New("ack failed") }
			}
			env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{Type: kind, GuildID: guild, Member: member, Data: data}})
			if !assert.Len(t, env.Responder.responses, 1) {
				return
			}
			if test.ackFails {
				assert.Empty(t, env.Responder.edits)
				assert.Zero(t, checks)
				return
			}
			if test.admin {
				assert.Equal(t, discordgo.InteractionResponseModal, env.Responder.responses[0].response.Type)
				return
			}
			if test.submit {
				if assert.Len(t, env.Responder.edits, 1) {
					assert.Contains(t, *env.Responder.edits[0].response.Content, test.want)
				}
			} else {
				assert.Contains(t, env.Responder.responses[0].response.Data.Content, test.want)
			}
			if test.guild != "" || test.testMode || test.missingMember {
				assert.Zero(t, checks)
			}
		})
	}
}

// Both command families must use controllers supplied by the composition root,
// including their responder, rather than constructing replacements inside Bot.
func TestWorkflow_BatchCommandsUseInjectedControllers(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, input, route string
	}{
		{name: "problem", input: `[{"class":"basicos","name":"Two Sum","url":"https://example.com"}]`, route: discord.ProblemBatchModalCustomID},
		{name: "ac", input: `[{"user_id":"123","problem_id":42}]`, route: discord.ACBatchModalCustomID},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			env.Discord.isMentorFunc = func(_, _ string) (bool, error) { return true, nil }
			client := &recordingDiscordInteractionClient{t: t}
			responder := dispatcher.NewDiscordInteractionResponder(client)
			if test.name == "problem" {
				env.ProblemController = mentor.NewProblemInteractionController(env.Services, env.Discord, responder)
			} else {
				env.ACController = mentor.NewACInteractionController(env.Discord, responder)
			}
			member := &discordgo.Member{User: &discordgo.User{ID: "mentor"}}
			env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand, GuildID: testGuildID, Member: member,
				Data: discordgo.ApplicationCommandInteractionData{Name: test.name, Options: []*discordgo.ApplicationCommandInteractionDataOption{{Name: "add_batch", Type: discordgo.ApplicationCommandOptionSubCommand}}},
			}})
			if !assert.Len(t, client.responses, 1) {
				return
			}
			assert.Equal(t, test.route, client.responses[0].response.Data.CustomID)
			env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionModalSubmit, GuildID: testGuildID, Member: member,
				Data: discordgo.ModalSubmitInteractionData{CustomID: test.route, Components: []discordgo.MessageComponent{
					&discordgo.Label{Component: &discordgo.TextInput{CustomID: "json", Value: test.input}},
					&discordgo.Label{Component: &discordgo.FileUpload{CustomID: "file"}},
				}},
			}})
			assert.Len(t, client.responses, 2)
			if assert.Len(t, client.edits, 1) {
				assert.Contains(t, *client.edits[0].response.Content, "Entries: 1")
			}
			assert.Empty(t, env.Responder.responses)
			assert.Empty(t, env.Responder.edits)
		})
	}
}
