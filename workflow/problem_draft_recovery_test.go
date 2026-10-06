package workflow

import (
	"fmt"
	"strings"
	"testing"

	appbot "github.com/JorgeIba/Hurion/app/bot"
	"github.com/JorgeIba/Hurion/app/controllers/mentor"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	form "github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflow_ExplicitControllerPreservesDraftAcrossBotInstances(t *testing.T) {
	t.Parallel()
	cfg := appbot.BotConfig{GuildID: testGuildID}
	env := setupWorkflowEnv(t, cfg)
	env.Discord.isMentorFunc = func(_, _ string) (bool, error) { return true, nil }
	env.seedClass(t, "basicos_1", "Basicos")
	responder := dispatcher.NewDiscordInteractionResponder(env.Responder)
	env.ProblemController = mentor.NewProblemInteractionController(env.Services, env.Discord, responder)
	env.Bot = appbot.NewBot(env.Services, env.Discord, env.Dispatcher, responder, cfg)

	failed := env.openProblemForm(t, "mentor_1", "basicos_1")
	failed.Name, failed.URL = "Recover through injected controller", "invalid"
	env.submitProblemForm(t, "mentor_1", failed)

	env.Bot = appbot.NewBot(env.Services, env.Discord, env.Dispatcher, responder, cfg)
	recovered := env.openProblemForm(t, "mentor_1", "basicos_1")
	assert.Equal(t, failed, recovered)
	recovered.URL = "https://example.com/problem"
	env.submitProblemForm(t, "mentor_1", recovered)
	assert.Empty(t, env.openProblemForm(t, "mentor_1", "basicos_1").Name, "successful delivery consumes the recovered draft")
}

func TestWorkflow_FailedProblemFormRecoversRawValuesAndDiscardsDraftOnCancelOrValidSubmit(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		validRetry bool
	}{
		{name: "closing the recovered form discards its draft"},
		{name: "a valid retry leaves the next form empty", validRetry: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			env.Discord.isMentorFunc = func(_, _ string) (bool, error) { return true, nil }
			classID := env.seedClass(t, "basicos_1", "Basicos")
			env.seedClassroom(t, testChannelID, "basicos_1")
			opened := env.openProblemForm(t, "mentor_1", "")
			assert.Empty(t, opened.Name)
			failed := form.CreateProblemModal{
				ClassCode: "basicos_1",
				ProblemModalFields: form.ProblemModalFields{
					Name: " Two Sum 🚀 ", URL: "ftp://example.com/problem", Difficulty: " HARD ", Priority: " Recommended ", Topic: " Binary search ",
				},
			}
			env.submitProblemForm(t, "mentor_1", failed)
			assert.Contains(t, *env.Responder.edits[len(env.Responder.edits)-1].response.Content, "URL must be")
			assert.Contains(t, *env.Responder.edits[len(env.Responder.edits)-1].response.Content, "Try /problem add class:basicos_1 again.")
			assert.Equal(t, failed, env.openProblemForm(t, "mentor_1", "basicos_1"), "explicit class recovers the channel-derived draft without normalizing values")

			// A corrected submission that still fails replaces the original input.
			failed.URL = "https://example.com/problem"
			failed.Difficulty = "extreme"
			env.submitProblemForm(t, "mentor_1", failed)
			assert.Contains(t, *env.Responder.edits[len(env.Responder.edits)-1].response.Content, "invalid difficulty")
			assert.Equal(t, failed, env.openProblemForm(t, "mentor_1", ""), "the latest failed values are recovered")
			if test.validRetry {
				failed.Difficulty = "Hard"
				env.submitProblemForm(t, "mentor_1", failed)
				assert.Contains(t, *env.Responder.edits[len(env.Responder.edits)-1].response.Content, "was added to class")
			}
			// Closing a modal sends no event, so cancellation is simulated by opening again without submitting.
			next := env.openProblemForm(t, "mentor_1", "")
			assert.Empty(t, next.Name)
			assert.Empty(t, next.URL)
			assert.Empty(t, next.Topic)
			problems, err := env.DB.GetProblemsByClass(classID)
			require.NoError(t, err)
			if test.validRetry {
				require.Len(t, problems, 1)
				problem := problems[0]
				assert.Positive(t, problem.ID)
				assert.Equal(t, classID, problem.ClassID)
				assert.Equal(t, "Two Sum 🚀", problem.Name)
				assert.Equal(t, "https://example.com/problem", problem.URL)
				require.NotNil(t, problem.Difficulty)
				require.NotNil(t, problem.Priority)
				require.NotNil(t, problem.Topic)
				assert.Equal(t, "hard", *problem.Difficulty)
				assert.Equal(t, "recommended", *problem.Priority)
				assert.Equal(t, "Binary search", *problem.Topic)
			} else {
				assert.Empty(t, problems, "cancelled forms do not save a problem")
			}
		})
	}
}

func TestWorkflow_ProblemDraftRetryInstructionRecoversTheSelectedClass(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name         string
		channelClass string
		optionClass  string
		wantClass    string
	}{
		{name: "class resolved from the channel", channelClass: "basicos_1", wantClass: "basicos_1"},
		{name: "explicit class differs from the channel", channelClass: "basicos_1", optionClass: "basicos_2", wantClass: "basicos_2"},
		{name: "explicit class from an unmapped channel", optionClass: "basicos_2", wantClass: "basicos_2"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			env.Discord.isMentorFunc = func(_, _ string) (bool, error) { return true, nil }
			env.seedClass(t, "basicos_1", "Basicos 1")
			env.seedClass(t, "basicos_2", "Basicos 2")
			if test.channelClass != "" {
				env.seedClassroom(t, testChannelID, test.channelClass)
			}
			failed := env.openProblemForm(t, "mentor_1", test.optionClass)
			require.Equal(t, test.wantClass, failed.ClassCode)
			failed.Name, failed.URL = " Two Sum ", "invalid"
			env.submitProblemForm(t, "mentor_1", failed)

			content := *env.Responder.edits[len(env.Responder.edits)-1].response.Content
			_, retryClass, found := strings.Cut(content, ". Try /problem add class:")
			require.True(t, found, "the retry instruction must identify the class used by the failed form")
			require.True(t, strings.HasSuffix(retryClass, " again."))
			retryClass = strings.TrimSuffix(retryClass, " again.")
			require.Equal(t, test.wantClass, retryClass)
			recovered := env.openProblemForm(t, "mentor_1", retryClass)
			assert.Equal(t, failed, recovered, "following the instruction must restore all original form values")
			recovered.URL = "https://example.com/problem"
			env.submitProblemForm(t, "mentor_1", recovered)

			for _, classCode := range []string{"basicos_1", "basicos_2"} {
				class, err := env.DB.GetClassByClassCode(classCode)
				require.NoError(t, err)
				problems, err := env.DB.GetProblemsByClass(class.ID)
				require.NoError(t, err)
				if classCode != test.wantClass {
					assert.Empty(t, problems, "retry must not create a problem in another class")
					continue
				}
				require.Len(t, problems, 1)
				assert.Equal(t, class.ID, problems[0].ClassID)
				assert.Equal(t, "Two Sum", problems[0].Name)
				assert.Equal(t, "https://example.com/problem", problems[0].URL)
			}
		})
	}
}

func TestWorkflow_ProblemDraftCannotBeRecoveredByAnotherUserOrClassOrAfterPermissionRevocation(t *testing.T) {
	t.Parallel()
	env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
	allowed := true
	env.Discord.isMentorFunc = func(_, _ string) (bool, error) { return allowed, nil }
	env.seedClass(t, "basicos_1", "Basicos")
	env.seedClass(t, "basicos_2", "Basicos 2")
	failed := env.openProblemForm(t, "mentor_1", "basicos_1")
	failed.Name, failed.URL = "My draft", "invalid"
	env.submitProblemForm(t, "mentor_1", failed)
	assert.Empty(t, env.openProblemForm(t, "mentor_2", "basicos_1").Name)
	assert.Empty(t, env.openProblemForm(t, "mentor_1", "basicos_2").Name)
	allowed = false
	env.handleInteractionEvent(addProblemCommand("mentor_1", "basicos_1"))
	assert.Contains(t, env.Responder.responses[len(env.Responder.responses)-1].response.Data.Content, "mentor or admin")
	allowed = true
	assert.Equal(t, failed, env.openProblemForm(t, "mentor_1", "basicos_1"), "denied access must not consume the draft")
}

func TestWorkflow_ProblemDraftsRemainIsolatedBetweenBotInstances(t *testing.T) {
	t.Parallel()
	first := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
	second := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
	for _, env := range []*workflowEnv{first, second} {
		env.Discord.isMentorFunc = func(_, _ string) (bool, error) { return true, nil }
		env.seedClass(t, "basicos_1", "Basicos")
	}
	for _, test := range []struct {
		env  *workflowEnv
		name string
	}{
		{env: first, name: "First bot draft"},
		{env: second, name: "Second bot draft"},
	} {
		modal := test.env.openProblemForm(t, "mentor_1", "basicos_1")
		modal.Name, modal.URL = test.name, "invalid"
		test.env.submitProblemForm(t, "mentor_1", modal)
	}
	assert.Equal(t, "First bot draft", first.openProblemForm(t, "mentor_1", "basicos_1").Name)
	assert.Equal(t, "Second bot draft", second.openProblemForm(t, "mentor_1", "basicos_1").Name)
}

func TestWorkflow_FailedModalDeliveryKeepsProblemDraftForNextCommand(t *testing.T) {
	t.Parallel()
	env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
	env.Discord.isMentorFunc = func(_, _ string) (bool, error) { return true, nil }
	env.seedClass(t, "basicos_1", "Basicos")
	failed := env.openProblemForm(t, "mentor_1", "basicos_1")
	failed.Name, failed.URL = "My original draft", "invalid"
	env.submitProblemForm(t, "mentor_1", failed)
	attempts := 0
	env.Responder.respondFunc = func(response *discordgo.InteractionResponse) error {
		require.Equal(t, discordgo.InteractionResponseModal, response.Type)
		attempts++
		return fmt.Errorf("Discord response failed")
	}
	env.handleInteractionEvent(addProblemCommand("mentor_1", "basicos_1"))
	assert.Equal(t, 1, attempts)
	env.Responder.respondFunc = nil
	recovered := env.openProblemForm(t, "mentor_1", "basicos_1")
	assert.Equal(t, failed, recovered)
	recovered.URL = "https://example.com/problem"
	env.submitProblemForm(t, "mentor_1", recovered)
	assert.Empty(t, env.openProblemForm(t, "mentor_1", "basicos_1").Name, "successful delivery consumed the draft")
}

func addProblemCommand(userID, classCode string) *discordgo.InteractionCreate {
	var options []*discordgo.ApplicationCommandInteractionDataOption
	if classCode != "" {
		options = append(options, &discordgo.ApplicationCommandInteractionDataOption{Name: "class", Type: discordgo.ApplicationCommandOptionString, Value: classCode})
	}
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		GuildID: testGuildID, ChannelID: testChannelID, Member: &discordgo.Member{User: &discordgo.User{ID: userID}},
		Type: discordgo.InteractionApplicationCommand,
		Data: discordgo.ApplicationCommandInteractionData{Name: "problem", Options: []*discordgo.ApplicationCommandInteractionDataOption{{Name: "add", Type: discordgo.ApplicationCommandOptionSubCommand, Options: options}}},
	}}
}

func (env *workflowEnv) openProblemForm(t *testing.T, userID, classCode string) form.CreateProblemModal {
	t.Helper()
	before := len(env.Responder.responses)
	env.handleInteractionEvent(addProblemCommand(userID, classCode))
	require.Len(t, env.Responder.responses, before+1)
	response := env.Responder.responses[before].response
	require.Equal(t, discordgo.InteractionResponseModal, response.Type)
	modal, err := form.ParseCreateProblemModal(submittedProblemFormData(t, response))
	require.NoError(t, err)
	return modal
}

func (env *workflowEnv) submitProblemForm(t *testing.T, userID string, modal form.CreateProblemModal) {
	t.Helper()
	response, err := form.BuildCreateProblemModalResponse(modal)
	require.NoError(t, err)
	before := len(env.Responder.responses)
	beforeEdits := len(env.Responder.edits)
	env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		GuildID: testGuildID, ChannelID: testChannelID, Member: &discordgo.Member{User: &discordgo.User{ID: userID}},
		Type: discordgo.InteractionModalSubmit, Data: submittedProblemFormData(t, response),
	}})
	require.Len(t, env.Responder.responses, before+1)
	assert.Equal(t, discordgo.InteractionResponseDeferredChannelMessageWithSource, env.Responder.responses[before].response.Type)
	assert.Equal(t, discordgo.MessageFlagsEphemeral, env.Responder.responses[before].response.Data.Flags)
	require.Len(t, env.Responder.edits, beforeEdits+1)
}

func submittedProblemFormData(t *testing.T, response *discordgo.InteractionResponse) discordgo.ModalSubmitInteractionData {
	t.Helper()
	data := discordgo.ModalSubmitInteractionData{CustomID: response.Data.CustomID}
	for _, component := range response.Data.Components {
		label, ok := component.(discordgo.Label)
		require.True(t, ok)
		input, ok := label.Component.(discordgo.TextInput)
		require.True(t, ok)
		data.Components = append(data.Components, &discordgo.Label{Component: &input})
	}
	return data
}
