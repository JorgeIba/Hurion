package workflow

import (
	"fmt"
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"strings"
	"testing"

	appbot "github.com/JorgeIba/Hurion/app/bot"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/services/mock_services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func (env *workflowEnv) openProblemEditor(t *testing.T, user string, id int64, class string) *discordgo.InteractionResponse {
	t.Helper()
	options := []*discordgo.ApplicationCommandInteractionDataOption{{Name: "problem", Type: discordgo.ApplicationCommandOptionString, Value: fmt.Sprint(id)}}
	if class != "" {
		options = append(options, &discordgo.ApplicationCommandInteractionDataOption{Name: "class", Type: discordgo.ApplicationCommandOptionString, Value: class})
	}
	env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		GuildID: testGuildID, ChannelID: testChannelID, Member: &discordgo.Member{User: &discordgo.User{ID: user}}, Type: discordgo.InteractionApplicationCommand,
		Data: discordgo.ApplicationCommandInteractionData{Name: "problem", Options: []*discordgo.ApplicationCommandInteractionDataOption{{Name: "edit", Type: discordgo.ApplicationCommandOptionSubCommand, Options: options}}},
	}})
	response := env.Responder.responses[len(env.Responder.responses)-1].response
	require.Equal(t, discordgo.InteractionResponseModal, response.Type)
	return response
}

func editorFields(t *testing.T, response *discordgo.InteractionResponse) interactiontransport.ProblemModalFields {
	t.Helper()
	modal, err := interactiontransport.ParseEditProblemModal(submittedProblemFormData(t, response))
	require.NoError(t, err)
	return modal.ProblemModalFields
}

func (env *workflowEnv) submitProblemEditor(t *testing.T, user string, response *discordgo.InteractionResponse, fields interactiontransport.ProblemModalFields) {
	t.Helper()
	modal, err := interactiontransport.ParseEditProblemModal(submittedProblemFormData(t, response))
	require.NoError(t, err)
	// Preserve IDs from the opened form while replacing only its editable values.
	submission, err := buildEditProblemModalForTest(interactiontransport.EditProblemModal{ProblemID: modal.ProblemID, ClassCode: modal.ClassCode, ProblemModalFields: fields})
	require.NoError(t, err)
	env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		GuildID: testGuildID, ChannelID: testChannelID, Member: &discordgo.Member{User: &discordgo.User{ID: user}}, Type: discordgo.InteractionModalSubmit,
		Data: submittedProblemFormData(t, submission),
	}})
}

func TestWorkflow_EditProblemResolvesDestinationCodeAtSubmit(t *testing.T) {
	t.Parallel()
	for _, reuseCode := range []bool{false, true} {
		t.Run(fmt.Sprintf("reuse code=%t", reuseCode), func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			env.Discord.isMentorFunc = func(_, _ string) (bool, error) { return true, nil }
			source := env.seedClass(t, "source", "Source")
			code := "destination|á%"
			destination := env.seedClass(t, code, "Destination")
			id := env.seedProblem(t, "Original", "https://example.com/original", source, "hard", "Trees")
			response := env.openProblemEditor(t, "mentor", id, code)
			require.NoError(t, env.DB.DeleteClass(destination))
			if reuseCode {
				destination = env.seedClass(t, code, "Replacement")
			}
			env.submitProblemEditor(t, "mentor", response, interactiontransport.ProblemModalFields{Name: "Changed", URL: "https://example.com/changed"})
			problem, err := env.DB.GetProblemByID(id)
			require.NoError(t, err)
			if reuseCode {
				assert.Equal(t, destination, problem.ClassID)
				assert.Equal(t, "Changed", problem.Name)
			} else {
				assert.Equal(t, source, problem.ClassID, "missing destination must not fall back to the current class")
				assert.Equal(t, "Original", problem.Name)
			}
		})
	}
}

func TestWorkflow_EditProblemMovesAndClearsWithoutLosingIdentityOrSolutions(t *testing.T) {
	t.Parallel()
	env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
	env.Discord.isMentorFunc = func(_, _ string) (bool, error) { return true, nil }
	source := env.seedClass(t, "source", "Source")
	destination := env.seedClass(t, "destination", "Destination")
	env.seedClassroom(t, testChannelID, "destination")
	id, err := env.DB.AddProblem(data.Problem{
		Name: "Original", URL: "https://example.com/original", ClassID: source,
		Difficulty: new("hard"), Topic: new("Trees"), Priority: new(service_model.PriorityRecommended),
	})
	require.NoError(t, err)
	env.seedUser(t, "student", "Student")
	require.NoError(t, env.DB.AddAC("student", id))
	before, err := env.DB.GetProblemByID(id)
	require.NoError(t, err)
	require.NotNil(t, before.Priority)
	assert.Equal(t, service_model.PriorityRecommended, *before.Priority)
	solvedBefore, err := env.DB.GetSolvedProblemsByUserAndClass("student", source)
	require.NoError(t, err)
	response := env.openProblemEditor(t, "mentor", id, "")
	assert.Contains(t, response.Data.Title, "source", "default class comes from the problem, not the channel")
	fields := editorFields(t, response)
	assert.Equal(t, "Original", fields.Name)
	assert.Equal(t, new("hard"), fields.Difficulty)
	assert.Equal(t, new("recommended"), fields.Priority)
	response = env.openProblemEditor(t, "mentor", id, "destination")
	env.submitProblemEditor(t, "mentor", response, interactiontransport.ProblemModalFields{
		Name: " New @everyone ", URL: " https://example.com/new ",
	})
	after, err := env.DB.GetProblemByID(id)
	require.NoError(t, err)
	assert.Equal(t, id, after.ID)
	assert.Equal(t, before.CreatedAt, after.CreatedAt)
	assert.Equal(t, destination, after.ClassID)
	assert.Equal(t, "New @everyone", after.Name)
	assert.Nil(t, after.Difficulty)
	assert.Nil(t, after.Priority)
	assert.Nil(t, after.Topic)
	solvedAfter, err := env.DB.GetSolvedProblemsByUserAndClass("student", destination)
	require.NoError(t, err)
	assert.Equal(t, solvedBefore, solvedAfter)
	receipt := env.Responder.edits[len(env.Responder.edits)-1].response
	assert.Contains(t, *receipt.Content, "was updated")
	require.NotNil(t, receipt.AllowedMentions)
	assert.Empty(t, receipt.AllowedMentions.Parse)
}

func TestWorkflow_EditProblemPreservesUnchangedMetadataSelections(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		difficulty *string
		priority   *string
	}{
		{name: "assigned metadata", difficulty: new("hard"), priority: new(service_model.PriorityRecommended)},
		{name: "not required priority", difficulty: new("medium"), priority: new(service_model.PriorityNotRequired)},
		{name: "unassigned metadata"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			env.Discord.isMentorFunc = func(_, _ string) (bool, error) { return true, nil }
			classID := env.seedClass(t, "source", "Source")
			id, err := env.DB.AddProblem(data.Problem{
				Name: "Original", URL: "https://example.com/original", ClassID: classID,
				Difficulty: test.difficulty, Priority: test.priority,
			})
			require.NoError(t, err)
			response := env.openProblemEditor(t, "mentor", id, "")
			fields := editorFields(t, response)
			fields.Name = "Changed"
			env.submitProblemEditor(t, "mentor", response, fields)
			problem, err := env.DB.GetProblemByID(id)
			require.NoError(t, err)
			assert.Equal(t, "Changed", problem.Name)
			assert.Equal(t, test.difficulty, problem.Difficulty)
			assert.Equal(t, test.priority, problem.Priority)
		})
	}
}

func TestWorkflow_EditProblemRecoversDraftForSameProblemAndDestination(t *testing.T) {
	t.Parallel()
	env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
	env.Discord.isMentorFunc = func(_, _ string) (bool, error) { return true, nil }
	source := env.seedClass(t, "source", "Source")
	destination := env.seedClass(t, "destination", "Destination")
	id := env.seedProblem(t, "Original", "https://example.com/original", source, "hard", "Trees")
	response := env.openProblemEditor(t, "mentor", id, "destination")
	fields := interactiontransport.ProblemModalFields{Name: " Raw name ", URL: " invalid ", Difficulty: new("hard"), Priority: new("recommended"), Topic: new(" Raw topic ")}
	env.submitProblemEditor(t, "mentor", response, fields)
	assert.Contains(t, *env.Responder.edits[len(env.Responder.edits)-1].response.Content, "class:destination` again")
	assert.Equal(t, "Original", editorFields(t, env.openProblemEditor(t, "mentor", id, "")).Name, "another destination has a separate draft")
	assert.Equal(t, "Original", editorFields(t, env.openProblemEditor(t, "other", id, "destination")).Name, "another actor has a separate draft")
	env.Responder.respondFunc = func(r *discordgo.InteractionResponse) error {
		if r.Type == discordgo.InteractionResponseModal {
			return fmt.Errorf("delivery failed")
		}
		return nil
	}
	env.openProblemEditor(t, "mentor", id, "destination")
	env.Responder.respondFunc = nil
	recovered := env.openProblemEditor(t, "mentor", id, "destination")
	assert.Equal(t, fields, editorFields(t, recovered), "failed delivery preserves text and selected metadata")
	assert.Equal(t, "Original", editorFields(t, env.openProblemEditor(t, "mentor", id, "destination")).Name, "successful opening consumes the draft, as in add")
	fields.URL = "https://example.com/repaired"
	env.submitProblemEditor(t, "mentor", recovered, fields)
	current, err := env.DB.GetProblemByID(id)
	require.NoError(t, err)
	assert.Equal(t, destination, current.ClassID)
	assert.Equal(t, "Raw name", current.Name)
	assert.Equal(t, "recommended", *current.Priority)
}

func TestWorkflow_EditProblemRetryIncludesResolvedClass(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		class     string
		longError bool
	}{
		{"class omitted", "", false},
		{"class explicit", "destination", false},
		{"class omitted with long error", "", true},
		{"class explicit with long error", "destination", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			env.Discord.isMentorFunc = func(_, _ string) (bool, error) { return true, nil }
			source := env.seedClass(t, "source", "Source")
			env.seedClass(t, "destination", "Destination")
			id := env.seedProblem(t, "Original", "https://example.com/original", source, "hard", "Trees")
			response := env.openProblemEditor(t, "mentor", id, test.class)
			original := env.Services.Problem
			if test.longError {
				mock := mock_services.NewMockProblemService(gomock.NewController(t))
				mock.EXPECT().ReplaceProblemByProblemID(gomock.Any()).Return(serviceerr.NewValidationError(strings.Repeat("x", 2100), nil))
				env.Services.Problem = mock
			}
			fields := interactiontransport.ProblemModalFields{Name: " Raw name ", URL: "invalid", Topic: new(" Raw topic ")}
			env.submitProblemEditor(t, "mentor", response, fields)
			env.Services.Problem = original
			classCode := test.class
			if classCode == "" {
				classCode = "source"
			}
			content := *env.Responder.edits[len(env.Responder.edits)-1].response.Content
			assert.Contains(t, content, fmt.Sprintf("/problem edit problem:%d class:%s", id, classCode))
			reopened := env.openProblemEditor(t, "mentor", id, classCode)
			assert.Equal(t, fields, editorFields(t, reopened))
		})
	}
}

func TestWorkflow_EditProblemRejectsRoleLoss(t *testing.T) {
	t.Parallel()
	env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
	allowed := true
	env.Discord.isMentorFunc = func(_, _ string) (bool, error) { return allowed, nil }
	class := env.seedClass(t, "source", "Source")
	id := env.seedProblem(t, "Original", "https://example.com/original", class, "hard", "Trees")
	response := env.openProblemEditor(t, "mentor", id, "")
	fields := interactiontransport.ProblemModalFields{Name: "Changed", URL: "https://example.com/new"}
	allowed = false
	env.submitProblemEditor(t, "mentor", response, fields)
	assert.Contains(t, *env.Responder.edits[len(env.Responder.edits)-1].response.Content, "mentor or admin")
	current, err := env.DB.GetProblemByID(id)
	require.NoError(t, err)
	assert.Equal(t, "Original", current.Name)
}

func TestWorkflow_EditProblemReceiptFailureDoesNotUndoWrite(t *testing.T) {
	t.Parallel()
	env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
	env.Discord.isMentorFunc = func(_, _ string) (bool, error) { return true, nil }
	class := env.seedClass(t, "source", "Source")
	id := env.seedProblem(t, "Original", "https://example.com/original", class, "hard", "Trees")
	before, err := env.DB.GetProblemByID(id)
	require.NoError(t, err)
	response := env.openProblemEditor(t, "mentor", id, "")
	problemService := env.Services.Problem
	mock := mock_services.NewMockProblemService(gomock.NewController(t))
	mock.EXPECT().ReplaceProblemByProblemID(gomock.Any()).DoAndReturn(func(params service_model.ReplaceProblemParams) error {
		return problemService.ReplaceProblemByProblemID(params)
	}).Times(1)
	env.Services.Problem = mock
	receiptAttempts := 0
	env.Responder.editFunc = func(*discordgo.WebhookEdit) error {
		receiptAttempts++
		current, err := env.DB.GetProblemByID(id)
		require.NoError(t, err)
		assert.Equal(t, "Changed", current.Name, "the write completes before sending the receipt")
		return fmt.Errorf("receipt failed")
	}
	fields := interactiontransport.ProblemModalFields{Name: "Changed", URL: "https://example.com/new"}
	env.submitProblemEditor(t, "mentor", response, fields)
	assert.Equal(t, 1, receiptAttempts)
	after, err := env.DB.GetProblemByID(id)
	require.NoError(t, err)
	assert.Equal(t, "Changed", after.Name)
	assert.Equal(t, "https://example.com/new", after.URL)
	assert.Equal(t, before.ID, after.ID)
	assert.Equal(t, before.CreatedAt, after.CreatedAt)
	assert.Equal(t, before.ClassID, after.ClassID)
	assert.Contains(t, *env.Responder.edits[len(env.Responder.edits)-1].response.Content, "was updated")
}

func TestWorkflow_EditProblemFailedAcknowledgementAndDeletedTarget(t *testing.T) {
	t.Parallel()
	env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
	env.Discord.isMentorFunc = func(_, _ string) (bool, error) { return true, nil }
	class := env.seedClass(t, "source", "Source")
	id := env.seedProblem(t, "Original", "https://example.com/original", class, "hard", "Trees")
	response := env.openProblemEditor(t, "mentor", id, "")
	env.Responder.respondFunc = func(*discordgo.InteractionResponse) error { return fmt.Errorf("ack failed") }
	fields := interactiontransport.ProblemModalFields{Name: "Changed", URL: "https://example.com/new"}
	env.submitProblemEditor(t, "mentor", response, fields)
	current, err := env.DB.GetProblemByID(id)
	require.NoError(t, err)
	assert.Equal(t, "Original", current.Name)
	env.Responder.respondFunc = nil
	_, err = env.DB.RemoveProblemByID(id)
	require.NoError(t, err)
	env.submitProblemEditor(t, "mentor", response, fields)
	_, err = env.DB.GetProblemByID(id)
	require.Error(t, err, "deleted target must not be recreated")
	assert.NotContains(t, *env.Responder.edits[len(env.Responder.edits)-1].response.Content, "was updated")
}

func TestWorkflow_DistinctProblemEditorsUseLastSuccessfulWrite(t *testing.T) {
	t.Parallel()
	env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
	env.Discord.isMentorFunc = func(_, _ string) (bool, error) { return true, nil }
	source := env.seedClass(t, "source", "Source")
	destination := env.seedClass(t, "destination", "Destination")
	id := env.seedProblem(t, "Original", "https://example.com/original", source, "hard", "Trees")
	older := env.openProblemEditor(t, "mentor1", id, "")
	newer := env.openProblemEditor(t, "mentor2", id, "destination")
	env.submitProblemEditor(t, "mentor2", newer, interactiontransport.ProblemModalFields{Name: "Newer", URL: "https://example.com/newer"})
	moved, err := env.DB.GetProblemByID(id)
	require.NoError(t, err)
	assert.Equal(t, destination, moved.ClassID)
	env.submitProblemEditor(t, "mentor1", older, editorFields(t, older))
	overwritten, err := env.DB.GetProblemByID(id)
	require.NoError(t, err)
	assert.Equal(t, "Original", overwritten.Name)
	assert.Equal(t, source, overwritten.ClassID)
}

func TestWorkflow_EditProblemRetainsDraftsForEveryFailedWrite(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		err  error
	}{
		{"validation", serviceerr.NewValidationError("invalid URL", nil)},
		{"duplicate rejected", serviceerr.NewDuplicateProblemError("duplicate problem", nil)},
		{"uncertain write", serviceerr.MakeGeneralError(fmt.Errorf("write could not be confirmed"))},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			env.Discord.isMentorFunc = func(_, _ string) (bool, error) { return true, nil }
			source := env.seedClass(t, "source", "Source")
			env.seedClass(t, "destination", "Destination")
			id := env.seedProblem(t, "Original", "https://example.com/original", source, "hard", "Trees")
			response := env.openProblemEditor(t, "mentor", id, "destination")
			mock := mock_services.NewMockProblemService(gomock.NewController(t))
			mock.EXPECT().ReplaceProblemByProblemID(gomock.Any()).Return(test.err).Times(1)
			original := env.Services.Problem
			env.Services.Problem = mock
			fields := interactiontransport.ProblemModalFields{Name: " Raw name ", URL: "https://example.com/new"}
			env.submitProblemEditor(t, "mentor", response, fields)
			env.Services.Problem = original
			reopened := env.openProblemEditor(t, "mentor", id, "destination")
			assert.Equal(t, fields, editorFields(t, reopened))
			assert.Contains(t, reopened.Data.Title, "destination")
		})
	}
}

func TestWorkflow_ProblemFormsPreserveOptionalAbsenceThroughPersistence(t *testing.T) {
	t.Parallel()
	for _, action := range []string{"add", "edit"} {
		for _, test := range []struct {
			name, difficulty, priority, topic       string
			wantDifficulty, wantPriority, wantTopic *string
		}{
			{name: "clear priority only", difficulty: "hard", priority: "unset", topic: "Graphs", wantDifficulty: new("hard"), wantTopic: new("Graphs")},
			{name: "clear difficulty only", difficulty: "unset", priority: "recommended", topic: "Graphs", wantPriority: new("recommended"), wantTopic: new("Graphs")},
			{name: "clear both metadata fields", difficulty: "unset", priority: "unset", topic: "Graphs", wantTopic: new("Graphs")},
			{name: "clear every optional field", difficulty: "unset", priority: "unset"},
			{name: "empty selections and blank topic", topic: " \t "},
			{name: "unset is ordinary topic text", difficulty: "hard", priority: "recommended", topic: "unset", wantDifficulty: new("hard"), wantPriority: new("recommended"), wantTopic: new("unset")},
		} {
			t.Run(action+"/"+test.name, func(t *testing.T) {
				t.Parallel()
				env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
				env.Discord.isMentorFunc = func(_, _ string) (bool, error) { return true, nil }
				classID := env.seedClass(t, "basicos_1", "Basicos")
				var response *discordgo.InteractionResponse
				var problemID int64
				if action == "edit" {
					var err error
					problemID, err = env.DB.AddProblem(data.Problem{
						Name: "Original", URL: "https://example.com/original", ClassID: classID,
						Difficulty: new("hard"), Priority: new("recommended"), Topic: new("Trees"),
					})
					require.NoError(t, err)
					response = env.openProblemEditor(t, "mentor", problemID, "")
				} else {
					env.openProblemForm(t, "mentor", "basicos_1")
					response = env.Responder.responses[len(env.Responder.responses)-1].response
				}
				submission := submittedProblemFormDataWithValues(t, response, map[string]string{
					"name": "Two Sum", "url": "https://example.com/problem",
					"difficulty": test.difficulty, "priority": test.priority, "topic": test.topic,
				})
				var fields interactiontransport.ProblemModalFields
				if action == "edit" {
					modal, err := interactiontransport.ParseEditProblemModal(submission)
					require.NoError(t, err)
					fields = modal.ProblemModalFields
				} else {
					modal, err := interactiontransport.ParseCreateProblemModal(submission)
					require.NoError(t, err)
					fields = modal.ProblemModalFields
				}
				assert.Equal(t, test.wantDifficulty, fields.Difficulty)
				assert.Equal(t, test.wantPriority, fields.Priority)
				assert.Equal(t, test.wantTopic, fields.Topic)
				env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
					GuildID: testGuildID, ChannelID: testChannelID,
					Member: &discordgo.Member{User: &discordgo.User{ID: "mentor"}},
					Type:   discordgo.InteractionModalSubmit, Data: submission,
				}})
				problems, err := env.DB.GetProblemsByClass(classID)
				require.NoError(t, err)
				require.Len(t, problems, 1)
				if action == "edit" {
					assert.Equal(t, problemID, problems[0].ID)
				}
				assert.Equal(t, "Two Sum", problems[0].Name)
				assert.Equal(t, test.wantDifficulty, problems[0].Difficulty)
				assert.Equal(t, test.wantPriority, problems[0].Priority)
				assert.Equal(t, test.wantTopic, problems[0].Topic)
			})
		}
	}
}
