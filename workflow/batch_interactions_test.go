package workflow

import (
	"errors"
	"fmt"
	"github.com/JorgeIba/Hurion/app/services"
	"strings"
	"testing"

	appbot "github.com/JorgeIba/Hurion/app/bot"
	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
	"github.com/JorgeIba/Hurion/app/controllers/mentor"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

func TestWorkflow_BatchCommandsOpenFormsAndValidateWithoutWrites(t *testing.T) {
	t.Parallel()
	for _, test := range []struct{ name, command, input, want string }{
		{name: "problem JSON", command: "problem", input: `[{"class":"basicos","name":"Two Sum","url":"https://example.com","topic":"Arrays"}]`, want: "Entries: 1"},
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
			input := test.input
			var problemID int64
			if test.command == "ac" {
				problemID = env.seedProblem(t, "Existing", "https://example.com/existing", classID, "easy", "Arrays")
				env.seedClassAccessRole(t, "basicos", "student")
				env.setRoles("123", []string{"student"})
				input = strings.ReplaceAll(input, `"problem_id":42`, fmt.Sprintf(`"problem_id":%d`, problemID))
			}
			checks := 0
			env.setRoles("mentor", []string{"mentor_role"})
			env.Discord.fetchMemberFunc = func(guildID, userID string) (*discordgo.Member, error) {
				checks++
				return env.Discord.GetCachedMember(guildID, userID)
			}
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
			expectedRoute := string(discordcommands.ACBatch)
			if test.command == "problem" {
				expectedRoute = string(discordcommands.ProblemBatch)
			}
			assert.Equal(t, expectedRoute, response.Data.CustomID)
			env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionModalSubmit, GuildID: testGuildID, Member: member,
				Data: discordgo.ModalSubmitInteractionData{CustomID: response.Data.CustomID, Components: []discordgo.MessageComponent{
					&discordgo.Label{Component: &discordgo.TextInput{CustomID: "json", Value: input}},
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
			expectedChecks := 0
			if strings.HasPrefix(test.want, "Entries:") {
				expectedChecks++
			}
			assert.Equal(t, expectedChecks, checks, "check entry at open/submit and targets before preview")
			problems, err := env.DB.GetProblemsByClass(classID)
			assert.NoError(t, err)
			if test.command == "problem" {
				assert.Empty(t, problems, "validation must not create problems")
			} else {
				assert.Len(t, problems, 1, "validation must not change existing problems")
			}
			_, err = env.DB.GetUserByUserID("123")
			assert.Error(t, err, "intake must not create student records")
			solved, err := env.DB.HasUserSolvedProblem("123", problemID)
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
		{name: "mentor denied", command: "problem", denied: true, want: "No changes were applied"},
		{name: "AC denied", command: "ac", denied: true, want: "No changes were applied"},
		{name: "missing member", command: "ac", missingMember: true, want: "identified server member"},
		{name: "wrong server", command: "ac", guild: "other", want: "configured server"},
		{name: "test mode whitelist", command: "ac", testMode: true, want: "not enabled"},
		{name: "admin opens AC form", command: "ac", admin: true},
		{name: "admin opens problem form", command: "problem", admin: true},
		{name: "authority revoked before submit", command: "ac", denied: true, submit: true, want: "No changes were applied"},
		{name: "failed ack stops processing", command: "ac", submit: true, ackFails: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID, TestMode: test.testMode})
			checks := 0
			if !test.denied && !test.admin {
				env.setRoles("mentor", []string{"mentor_role"})
			}
			env.Discord.fetchMemberFunc = func(guildID, userID string) (*discordgo.Member, error) {
				checks++
				return env.Discord.GetCachedMember(guildID, userID)
			}
			if test.admin {
				env.setRoles("mentor", []string{"admin_role"})
			}
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
				classID := env.seedClass(t, "basicos", "Basicos")
				id := env.seedProblem(t, "Existing", "https://example.com/existing", classID, "easy", "Arrays")
				env.seedClassAccessRole(t, "basicos", "student")
				env.setRoles("123", []string{"student"})
				data = discordgo.ModalSubmitInteractionData{CustomID: string(discordcommands.ACBatch), Components: []discordgo.MessageComponent{
					&discordgo.Label{Component: &discordgo.TextInput{CustomID: "json", Value: fmt.Sprintf(`[{"user_id":"123","problem_id":%d}]`, id)}},
					&discordgo.Label{Component: &discordgo.FileUpload{CustomID: "file"}},
				}}
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
			if test.denied && !test.submit {
				assert.Equal(t, discordgo.InteractionResponseModal, env.Responder.responses[0].response.Type)
				assert.Zero(t, checks, "opening an untargeted form grants no permission")
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
		{name: "problem", input: `[{"class":"basicos","name":"Two Sum","url":"https://example.com","topic":"Arrays"}]`, route: string(discordcommands.ProblemBatch)},
		{name: "ac", input: `[{"user_id":"123","problem_id":42}]`, route: string(discordcommands.ACBatch)},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			env.setRoles("mentor", []string{"mentor_role"})
			classID := env.seedClass(t, "basicos", "Basicos")
			input := test.input
			if test.name == "ac" {
				id := env.seedProblem(t, "Existing", "https://example.com/existing", classID, "easy", "Arrays")
				env.seedClassAccessRole(t, "basicos", "student")
				env.setRoles("123", []string{"student"})
				input = strings.ReplaceAll(input, `"problem_id":42`, fmt.Sprintf(`"problem_id":%d`, id))
			}
			client := &recordingDiscordInteractionClient{t: t}
			responder := dispatcher.NewDiscordInteractionResponder(client)
			if test.name == "problem" {
				env.ProblemController = mentor.NewProblemInteractionController(env.Services, responder)
			} else {
				env.ACController = mentor.NewACInteractionController(env.Services, env.Discord, responder)
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
					&discordgo.Label{Component: &discordgo.TextInput{CustomID: "json", Value: input}},
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

func TestWorkflow_BatchValidationReportsDomainIssuesWithoutWrites(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, command, input                     string
		want                                     []string
		denyAccess, lookupFails, alreadyRecorded bool
		wantLookups                              int
	}{
		{name: "unknown class", command: "problem", input: `[{"class":"missing","name":"Valid","url":"https://example.com","topic":"Arrays"}]`, want: []string{"Batch validation failed", "Entry 1", "Class code `missing` not found"}},
		{name: "invalid problem values", command: "problem", input: `[{}]`, want: []string{"Entry 1", "Please correct the values"}},
		{name: "valid problem followed by unknown class", command: "problem", input: `[{"class":"basicos","name":"Valid","url":"https://example.com","topic":"Arrays"},{"class":"missing","name":"Other","url":"https://example.com","topic":"Arrays"}]`, want: []string{"Batch validation failed", "Entry 2"}},
		{name: "empty problem batch", command: "problem", input: `[]`, want: []string{"between 1 and 1000 operations"}},
		{name: "oversized problem batch", command: "problem", input: `[` + strings.Repeat(`{},`, 1000) + `{}]`, want: []string{"between 1 and 1000 operations"}},
		{name: "empty AC batch", command: "ac", input: `[]`, want: []string{"between 1 and 1000 operations"}},
		{name: "oversized AC batch", command: "ac", input: `[` + strings.Repeat(`{},`, 1000) + `{}]`, want: []string{"between 1 and 1000 operations"}},
		{name: "invalid student", command: "ac", input: `[{"user_id":"Ana","problem_id":1}]`, want: []string{"Entry 1", "positive numeric Discord ID"}},
		{name: "invalid problem ID", command: "ac", input: `[{"user_id":"123","problem_id":0}]`, want: []string{"Entry 1", "problem_id must be positive"}},
		{name: "unknown problem", command: "ac", input: `[{"user_id":"123","problem_id":9999}]`, want: []string{"Entry 1", "Problem id `9999` not found"}, wantLookups: 1},
		{name: "no class access", command: "ac", input: `[{"user_id":"123","problem_id":1}]`, denyAccess: true, want: []string{"Entry 1", "doesn't have access"}, wantLookups: 1},
		{name: "Discord lookup failure", command: "ac", input: `[{"user_id":"123","problem_id":1}]`, lookupFails: true, want: []string{"Could not resolve a batch student from Discord"}, wantLookups: 1},
		{name: "valid AC followed by unknown problem", command: "ac", input: `[{"user_id":"123","problem_id":1},{"user_id":"123","problem_id":9999}]`, want: []string{"Batch validation failed", "Entry 2"}, wantLookups: 1},
		{name: "existing AC remains valid", command: "ac", input: `[{"user_id":"123","problem_id":1}]`, alreadyRecorded: true, want: []string{"AC batch validated", "Entries: 1", "Already recorded entries: 1"}, wantLookups: 1},
		{name: "canonical student IDs resolve once", command: "ac", input: `[{"user_id":" 00123 ","problem_id":1},{"user_id":"123","problem_id":1}]`, alreadyRecorded: true, want: []string{"AC batch validated", "Entries: 2", "Students: 1", "Already recorded entries: 2"}, wantLookups: 1},
		{name: "repeated existing ACs preserved", command: "ac", input: `[{"user_id":"123","problem_id":1},{"user_id":"123","problem_id":1}]`, alreadyRecorded: true, want: []string{"AC batch validated", "Entries: 2", "Students: 1", "Already recorded entries: 2"}, wantLookups: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			env.setRoles("mentor", []string{"mentor_role"})
			classID := env.seedClass(t, "basicos", "Basicos")
			problemID := env.seedProblem(t, "Existing", "https://example.com/existing", classID, "easy", "Arrays")
			input := strings.ReplaceAll(test.input, `"problem_id":1}`, fmt.Sprintf(`"problem_id":%d}`, problemID))
			env.seedClassAccessRole(t, "basicos", "student")
			lookups := 0
			env.Discord.getCachedMemberFunc = func(_, userID string) (*discordgo.Member, error) {
				if userID == "mentor" {
					return &discordgo.Member{User: &discordgo.User{ID: userID}, Roles: []string{"mentor_role"}}, nil
				}
				lookups++
				if test.lookupFails {
					return nil, errors.New("private Discord cache failure")
				}
				roles := []string{"student"}
				if test.denyAccess {
					roles = nil
				}
				return &discordgo.Member{User: &discordgo.User{ID: userID}, Roles: roles}, nil
			}
			env.Discord.fetchMemberFunc = func(_, userID string) (*discordgo.Member, error) {
				if userID == "mentor" {
					return &discordgo.Member{User: &discordgo.User{ID: userID}, Roles: []string{"mentor_role"}}, nil
				}
				return nil, errors.New("private Discord HTTP failure")
			}
			if test.alreadyRecorded {
				env.seedUser(t, "123", "Student")
				assert.NoError(t, env.Services.Solution.GrantACWithoutEnrollmentCheck(services.UserActor("mentor"), "123", problemID))
			}
			member := &discordgo.Member{User: &discordgo.User{ID: "mentor"}}
			env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand, GuildID: testGuildID, Member: member,
				Data: discordgo.ApplicationCommandInteractionData{Name: test.command, Options: []*discordgo.ApplicationCommandInteractionDataOption{{Name: "add_batch", Type: discordgo.ApplicationCommandOptionSubCommand}}},
			}})
			if !assert.Len(t, env.Responder.responses, 1) {
				return
			}
			route := env.Responder.responses[0].response.Data.CustomID
			env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionModalSubmit, GuildID: testGuildID, Member: member,
				Data: discordgo.ModalSubmitInteractionData{CustomID: route, Components: []discordgo.MessageComponent{
					&discordgo.Label{Component: &discordgo.TextInput{CustomID: "json", Value: input}},
					&discordgo.Label{Component: &discordgo.FileUpload{CustomID: "file"}},
				}},
			}})
			if assert.Len(t, env.Responder.edits, 1) {
				content := *env.Responder.edits[0].response.Content
				for _, want := range test.want {
					assert.Contains(t, content, want)
				}
				assert.Contains(t, content, "No changes were applied")
				assert.NotContains(t, content, "private Discord")
				assert.NotContains(t, content, "Service Error")
				if !test.alreadyRecorded {
					assert.NotContains(t, content, "batch validated")
				}
			}
			assert.Equal(t, test.wantLookups, lookups)
			problems, err := env.DB.GetProblemsByClass(classID)
			assert.NoError(t, err)
			assert.Len(t, problems, 1)
			solved, err := env.DB.HasUserSolvedProblem("123", problemID)
			assert.NoError(t, err)
			assert.Equal(t, test.alreadyRecorded, solved)
			user, err := env.DB.GetUserByUserID("123")
			if test.alreadyRecorded {
				assert.NoError(t, err)
				assert.Equal(t, "Student", user.DisplayName)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestWorkflow_ACBatchRejectsInvalidInputBeforeStudentLookups(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, input string
		want        []string
		users       []string
	}{
		{
			name:  "valid entry followed by invalid student",
			input: `[{"user_id":"123","problem_id":1},{"user_id":"Ana","problem_id":1}]`,
			want:  []string{"Entry 2", "positive numeric Discord ID"},
			users: []string{"123", "Ana"},
		},
		{
			name:  "all invalid entries are reported",
			input: `[{"user_id":"Ana","problem_id":1},{"user_id":"123","problem_id":0}]`,
			want:  []string{"Entry 1", "positive numeric Discord ID", "Entry 2", "problem_id must be positive"},
			users: []string{"Ana", "123"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			env.setRoles("mentor", []string{"mentor_role"})
			classID := env.seedClass(t, "basicos", "Basicos")
			problemID := env.seedProblem(t, "Existing", "https://example.com/existing", classID, "easy", "Arrays")
			env.seedClassAccessRole(t, "basicos", "student")
			cachedLookups, httpLookups := 0, 0
			env.Discord.getCachedMemberFunc = func(_, userID string) (*discordgo.Member, error) {
				if userID == "mentor" {
					return &discordgo.Member{User: &discordgo.User{ID: userID}, Roles: []string{"mentor_role"}}, nil
				}
				cachedLookups++
				return nil, errors.New("unexpected student lookup")
			}
			env.Discord.fetchMemberFunc = func(_, userID string) (*discordgo.Member, error) {
				if userID == "mentor" {
					return &discordgo.Member{User: &discordgo.User{ID: userID}, Roles: []string{"mentor_role"}}, nil
				}
				httpLookups++
				return nil, errors.New("unexpected student lookup")
			}
			input := strings.ReplaceAll(test.input, `"problem_id":1}`, fmt.Sprintf(`"problem_id":%d}`, problemID))
			member := &discordgo.Member{User: &discordgo.User{ID: "mentor"}}
			env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand, GuildID: testGuildID, Member: member,
				Data: discordgo.ApplicationCommandInteractionData{Name: "ac", Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{Name: "add_batch", Type: discordgo.ApplicationCommandOptionSubCommand},
				}},
			}})
			if !assert.Len(t, env.Responder.responses, 1) {
				return
			}
			route := env.Responder.responses[0].response.Data.CustomID
			env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionModalSubmit, GuildID: testGuildID, Member: member,
				Data: discordgo.ModalSubmitInteractionData{CustomID: route, Components: []discordgo.MessageComponent{
					&discordgo.Label{Component: &discordgo.TextInput{CustomID: "json", Value: input}},
					&discordgo.Label{Component: &discordgo.FileUpload{CustomID: "file"}},
				}},
			}})
			assert.Zero(t, cachedLookups, "validate the entire input before consulting Discord's cache")
			assert.Zero(t, httpLookups, "invalid input must not fetch students over HTTP")
			assert.Len(t, env.Responder.responses, 2, "opening the form and acknowledging its submission are the only responses")
			if !assert.Len(t, env.Responder.edits, 1) {
				return
			}
			receipt := env.Responder.edits[0].response
			for _, want := range test.want {
				assert.Contains(t, *receipt.Content, want)
			}
			assert.Contains(t, *receipt.Content, "Batch validation failed")
			assert.Contains(t, *receipt.Content, "No changes were applied")
			if receipt.Components != nil {
				assert.Empty(t, *receipt.Components, "invalid input must not create a preview")
			}
			problems, err := env.DB.GetProblemsByClass(classID)
			assert.NoError(t, err)
			assert.Len(t, problems, 1)

			for _, userID := range test.users {
				_, err := env.DB.GetUserByUserID(userID)
				assert.Error(t, err, "invalid input must not create student %s", userID)
				solved, err := env.DB.HasUserSolvedProblem(userID, problemID)
				assert.NoError(t, err)
				assert.False(t, solved, "invalid input must not record an AC for %s", userID)
			}
		})
	}
}

func TestWorkflow_BatchPreviewAndCancellationApplyNoWrites(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, command string
		deliveryFails bool
	}{
		{name: "problem cancellation", command: "problem"},
		{name: "AC cancellation", command: "ac"},
		{name: "problem preview delivery failure", command: "problem", deliveryFails: true},
		{name: "AC preview delivery failure", command: "ac", deliveryFails: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			env.setRoles("mentor", []string{"mentor_role"})
			classID := env.seedClass(t, "basicos", "Basicos")
			input := `[{"class":"basicos","name":"New","url":"https://example.com/new","topic":"Arrays"}]`
			var problemID int64
			if test.command == "ac" {
				problemID = env.seedProblem(t, "Existing", "https://example.com/existing", classID, "easy", "Arrays")
				env.seedClassAccessRole(t, "basicos", "student")
				env.setRoles("123", []string{"student"})
				input = fmt.Sprintf(`[{"user_id":"123","problem_id":%d}]`, problemID)
			}
			member := &discordgo.Member{User: &discordgo.User{ID: "mentor"}}
			env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand, GuildID: testGuildID, Member: member,
				Data: discordgo.ApplicationCommandInteractionData{Name: test.command, Options: []*discordgo.ApplicationCommandInteractionDataOption{{Name: "add_batch", Type: discordgo.ApplicationCommandOptionSubCommand}}},
			}})
			if !assert.Len(t, env.Responder.responses, 1) {
				return
			}
			route := env.Responder.responses[0].response.Data.CustomID
			if test.deliveryFails {
				env.Responder.editFunc = func(*discordgo.WebhookEdit) error { return errors.New("receipt delivery failure") }
			}
			env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionModalSubmit, GuildID: testGuildID, Member: member,
				Data: discordgo.ModalSubmitInteractionData{CustomID: route, Components: []discordgo.MessageComponent{
					&discordgo.Label{Component: &discordgo.TextInput{CustomID: "json", Value: input}},
					&discordgo.Label{Component: &discordgo.FileUpload{CustomID: "file"}},
				}},
			}})
			if !assert.Len(t, env.Responder.edits, 1) {
				return
			}
			preview := env.Responder.edits[0].response
			assert.Contains(t, *preview.Content, "batch validated")
			if !assert.NotNil(t, preview.Components) || !assert.Len(t, *preview.Components, 1) {
				return
			}
			row := (*preview.Components)[0].(discordgo.ActionsRow)
			if !assert.Len(t, row.Components, 2) {
				return
			}
			button := row.Components[1].(discordgo.Button)
			assert.Equal(t, "Cancel", button.Label)
			assert.LessOrEqual(t, len(button.CustomID), 100)
			env.Responder.editFunc = nil
			cancel := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionMessageComponent, GuildID: testGuildID, Member: member,
				Data: discordgo.MessageComponentInteractionData{CustomID: button.CustomID, ComponentType: discordgo.ButtonComponent},
			}}
			env.handleInteractionEvent(cancel)
			if test.deliveryFails {
				assert.Len(t, env.Responder.edits, 1, "a failed preview delivery must discard the pending batch")
				if assert.Len(t, env.Responder.responses, 3) {
					assert.Equal(t, discord.BatchUnavailableMessage, env.Responder.responses[2].response.Data.Content)
				}
			} else {
				if assert.Len(t, env.Responder.responses, 3) {
					assert.Equal(t, discordgo.InteractionResponseDeferredMessageUpdate, env.Responder.responses[2].response.Type)
				}
				if assert.Len(t, env.Responder.edits, 2) {
					assert.Equal(t, "Batch cancelled. No changes were applied.", *env.Responder.edits[1].response.Content)
					assert.NotNil(t, env.Responder.edits[1].response.Components)
					assert.Empty(t, *env.Responder.edits[1].response.Components)
				}
				env.handleInteractionEvent(cancel)
				assert.Len(t, env.Responder.edits, 2, "repeated cancellation must not update the preview again")
				if assert.Len(t, env.Responder.responses, 4) {
					assert.Equal(t, discord.BatchUnavailableMessage, env.Responder.responses[3].response.Data.Content)
				}
			}
			problems, err := env.DB.GetProblemsByClass(classID)
			assert.NoError(t, err)
			if test.command == "problem" {
				assert.Empty(t, problems)
			} else {
				assert.Len(t, problems, 1)
			}
			_, err = env.DB.GetUserByUserID("123")
			assert.Error(t, err, "preview and cancellation must not create the student")
			if problemID != 0 {
				env.assertProblemNotSolved(t, "123", problemID)
			}
		})
	}
}

func TestWorkflow_BatchCancelButtonsHonorBotGates(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, guild, customID, want string
		testMode                    bool
	}{
		{name: "wrong server", guild: "other", customID: string(discordcommands.CancelProblemBatch) + "|id", want: "configured server"},
		{name: "development whitelist", guild: testGuildID, customID: string(discordcommands.CancelACBatch) + "|id", testMode: true, want: "not enabled"},
		{name: "unrelated button ignored", guild: testGuildID, customID: "unrelated|id"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID, TestMode: test.testMode})
			checks := 0
			env.setRoles("mentor", []string{"mentor_role"})
			env.Discord.fetchMemberFunc = func(guildID, userID string) (*discordgo.Member, error) {
				checks++
				return env.Discord.GetCachedMember(guildID, userID)
			}
			env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionMessageComponent, GuildID: test.guild, Member: &discordgo.Member{User: &discordgo.User{ID: "mentor"}},
				Data: discordgo.MessageComponentInteractionData{CustomID: test.customID, ComponentType: discordgo.ButtonComponent},
			}})
			assert.Zero(t, checks, "bot gates must run before management authorization")
			assert.Empty(t, env.Responder.edits)
			if test.want == "" {
				assert.Empty(t, env.Responder.responses)
			} else if assert.Len(t, env.Responder.responses, 1) {
				assert.Contains(t, env.Responder.responses[0].response.Data.Content, test.want)
				assert.Equal(t, discordgo.MessageFlagsEphemeral, env.Responder.responses[0].response.Data.Flags)
			}
		})
	}
}
