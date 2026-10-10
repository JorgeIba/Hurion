package workflow

import (
	"errors"
	"fmt"
	"testing"

	appbot "github.com/JorgeIba/Hurion/app/bot"
	"github.com/JorgeIba/Hurion/app/controllers/mentor"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func submitBatchPreview(t *testing.T, env *workflowEnv, command, input string) *discordgo.WebhookEdit {
	t.Helper()
	member := &discordgo.Member{User: &discordgo.User{ID: "mentor"}}
	env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionApplicationCommand, GuildID: testGuildID, Member: member,
		Data: discordgo.ApplicationCommandInteractionData{Name: command, Options: []*discordgo.ApplicationCommandInteractionDataOption{{Name: "add_batch", Type: discordgo.ApplicationCommandOptionSubCommand}}},
	}})
	response := env.Responder.responses[len(env.Responder.responses)-1].response
	require.Equal(t, discordgo.InteractionResponseModal, response.Type)
	env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionModalSubmit, GuildID: testGuildID, Member: member,
		Data: discordgo.ModalSubmitInteractionData{CustomID: response.Data.CustomID, Components: []discordgo.MessageComponent{
			&discordgo.Label{Component: &discordgo.TextInput{CustomID: "json", Value: input}},
			&discordgo.Label{Component: &discordgo.FileUpload{CustomID: "file"}},
		}},
	}})
	preview := env.Responder.edits[len(env.Responder.edits)-1].response
	require.Contains(t, *preview.Content, "batch validated")
	return preview
}

func batchButtonEvent(t *testing.T, preview *discordgo.WebhookEdit, label string) *discordgo.InteractionCreate {
	t.Helper()
	require.NotNil(t, preview.Components)
	row := (*preview.Components)[0].(discordgo.ActionsRow)
	for _, component := range row.Components {
		button := component.(discordgo.Button)
		if button.Label == label {
			return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionMessageComponent, GuildID: testGuildID, Member: &discordgo.Member{User: &discordgo.User{ID: "mentor"}},
				Data: discordgo.MessageComponentInteractionData{CustomID: button.CustomID, ComponentType: discordgo.ButtonComponent},
			}}
		}
	}
	t.Fatal("missing batch button: " + label)
	return nil
}

func TestWorkflow_BatchConfirmationWritesOnceAndReportsActualResults(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, command                            string
		repeated, alreadyRecorded, deliveryFails bool
	}{
		{name: "problems", command: "problem"},
		{name: "problem duplicates allowed", command: "problem", repeated: true},
		{name: "ACs", command: "ac"},
		{name: "AC repeats", command: "ac", repeated: true},
		{name: "existing AC", command: "ac", alreadyRecorded: true},
		{name: "problem result delivery failure", command: "problem", deliveryFails: true},
		{name: "AC result delivery failure", command: "ac", deliveryFails: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			env.Discord.isMentorFunc = func(string, string) (bool, error) { return true, nil }
			classID := env.seedClass(t, "basicos", "Basicos")
			entry := `{"class":"basicos","name":"New","url":"https://example.com/new"}`
			var problemID int64
			if test.command == "ac" {
				problemID = env.seedProblem(t, "Existing", "https://example.com/existing", classID, "easy", "Arrays")
				env.seedClassAccessRole(t, "basicos", "student")
				env.setRoles("123", []string{"student"})
				entry = fmt.Sprintf(`{"user_id":"123","problem_id":%d}`, problemID)
				if test.alreadyRecorded {
					require.NoError(t, env.Services.Solution.AddACToUserID("123", problemID))
				}
			}
			input := "[" + entry + "]"
			if test.repeated {
				input = "[" + entry + "," + entry + "]"
			}
			preview := submitBatchPreview(t, env, test.command, input)
			if test.deliveryFails {
				env.Responder.editFunc = func(*discordgo.WebhookEdit) error { return errors.New("delivery failed") }
			}
			event := batchButtonEvent(t, preview, "Confirm")
			env.handleInteractionEvent(event)
			require.Len(t, env.Responder.edits, 2)
			receipt := env.Responder.edits[1].response
			require.Empty(t, *receipt.Components)
			assert.Empty(t, receipt.AllowedMentions.Parse)
			assert.Contains(t, *receipt.Content, "Batch completed")
			assert.Equal(t, discordgo.InteractionResponseDeferredMessageUpdate, env.Responder.responses[2].response.Type)
			if test.command == "problem" {
				count := 1
				if test.repeated {
					count = 2
				}
				problems, err := env.DB.GetProblemsByClass(classID)
				require.NoError(t, err)
				require.Len(t, problems, count)
				assert.Contains(t, *receipt.Content, fmt.Sprintf("Problems created: %d", count))
				for _, problem := range problems {
					assert.Contains(t, *receipt.Content, fmt.Sprint(problem.ID))
				}
			} else {
				env.assertProblemSolved(t, "123", problemID)
				recorded, already := 1, 0
				if test.repeated {
					already = 1
				}
				if test.alreadyRecorded {
					recorded, already = 0, 1
				}
				assert.Contains(t, *receipt.Content, fmt.Sprintf("ACs recorded: %d. Already recorded: %d.", recorded, already))
			}
			env.Responder.editFunc = nil
			env.handleInteractionEvent(event)
			env.handleInteractionEvent(batchButtonEvent(t, preview, "Cancel"))
			assert.Len(t, env.Responder.edits, 2, "a consumed batch cannot be executed or cancelled again, including after delivery failure")
			assert.Equal(t, discord.BatchUnavailableMessage, env.Responder.responses[len(env.Responder.responses)-1].response.Data.Content)
		})
	}
}

func TestWorkflow_BatchConfirmationRevalidatesBeforeAnyWrite(t *testing.T) {
	t.Parallel()
	for _, command := range []string{"problem", "ac"} {
		t.Run(command, func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			env.Discord.isMentorFunc = func(string, string) (bool, error) { return true, nil }
			classID := env.seedClass(t, "basicos", "Basicos")
			otherClassID := env.seedClass(t, "other", "Other")
			input := `[{"class":"basicos","name":"Valid","url":"https://example.com/valid"},{"class":"other","name":"Invalid after preview","url":"https://example.com/other"}]`
			var problemID int64
			if command == "ac" {
				problemID = env.seedProblem(t, "Existing", "https://example.com/existing", classID, "easy", "Arrays")
				env.seedClassAccessRole(t, "basicos", "student")
				env.setRoles("123", []string{"student"})
				env.setRoles("456", []string{"student"})
				input = fmt.Sprintf(`[{"user_id":"123","problem_id":%d},{"user_id":"456","problem_id":%d}]`, problemID, problemID)
			}
			preview := submitBatchPreview(t, env, command, input)
			if command == "problem" {
				require.NoError(t, env.DB.DeleteClass(otherClassID))
			} else {
				env.setRoles("456", nil)
			}
			event := batchButtonEvent(t, preview, "Confirm")
			env.handleInteractionEvent(event)
			require.Len(t, env.Responder.edits, 2)
			receipt := env.Responder.edits[1].response
			assert.Contains(t, *receipt.Content, "Entry 2")
			assert.Contains(t, *receipt.Content, "No changes were applied")
			assert.Contains(t, *receipt.Content, "Submit the batch again")
			assert.Empty(t, *receipt.Components)
			problems, err := env.DB.GetProblemsByClass(classID)
			require.NoError(t, err)
			if command == "problem" {
				assert.Empty(t, problems)
			} else {
				env.assertProblemNotSolved(t, "123", problemID)
				for _, id := range []string{"123", "456"} {
					_, err := env.DB.GetUserByUserID(id)
					assert.Error(t, err, "revalidation must not create students")
				}
			}
			env.handleInteractionEvent(event)
			assert.Len(t, env.Responder.edits, 2)
		})
	}
}

// Fail the second write after applying it to exercise an uncertain storage outcome.
type uncertainBatchStore struct {
	data.DataStore
	writes int
}

func (store *uncertainBatchStore) AddProblem(problem data.Problem) (int64, error) {
	store.writes++
	id, err := store.DataStore.AddProblem(problem)
	if err == nil && store.writes == 2 {
		return 0, errors.New("lost acknowledgement")
	}
	return id, err
}
func (store *uncertainBatchStore) AddAC(userID string, problemID int64) error {
	store.writes++
	err := store.DataStore.AddAC(userID, problemID)
	if err == nil && store.writes == 2 {
		return errors.New("lost acknowledgement")
	}
	return err
}

func TestWorkflow_BatchStorageFailureReportsProgressWithoutRetrying(t *testing.T) {
	t.Parallel()
	for _, command := range []string{"problem", "ac"} {
		t.Run(command, func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			env.Discord.isMentorFunc = func(string, string) (bool, error) { return true, nil }
			classID := env.seedClass(t, "basicos", "Basicos")
			store := &uncertainBatchStore{DataStore: env.DB}
			svcs := services.NewServices(store)
			responder := dispatcher.NewDiscordInteractionResponder(env.Responder)
			env.ProblemController = mentor.NewProblemInteractionController(svcs, env.Discord, responder)
			env.ACController = mentor.NewACInteractionController(svcs, env.Discord, env.Discord, responder)
			input := `[{"class":"basicos","name":"One","url":"https://example.com/one"},{"class":"basicos","name":"Two","url":"https://example.com/two"},{"class":"basicos","name":"Three","url":"https://example.com/three"}]`
			var problemID int64
			if command == "ac" {
				problemID = env.seedProblem(t, "Existing", "https://example.com/existing", classID, "easy", "Arrays")
				env.seedClassAccessRole(t, "basicos", "student")
				for _, id := range []string{"123", "456", "789"} {
					env.setRoles(id, []string{"student"})
				}
				input = fmt.Sprintf(`[{"user_id":"123","problem_id":%d},{"user_id":"456","problem_id":%d},{"user_id":"789","problem_id":%d}]`, problemID, problemID, problemID)
			}
			preview := submitBatchPreview(t, env, command, input)
			event := batchButtonEvent(t, preview, "Confirm")
			env.handleInteractionEvent(event)
			require.Len(t, env.Responder.edits, 2)
			receipt := env.Responder.edits[1].response
			assert.Contains(t, *receipt.Content, "Batch stopped")
			assert.Contains(t, *receipt.Content, "may also have been applied")
			assert.NotContains(t, *receipt.Content, "No changes were applied")
			assert.Empty(t, *receipt.Components)
			assert.Equal(t, 2, store.writes, "stop subsequent operations")
			if command == "problem" {
				assert.Contains(t, *receipt.Content, "Confirmed problem creations: 1")
				problems, err := env.DB.GetProblemsByClass(classID)
				require.NoError(t, err)
				assert.Len(t, problems, 2)
			} else {
				assert.Contains(t, *receipt.Content, "Confirmed AC registrations: 1")
				env.assertProblemSolved(t, "123", problemID)
				env.assertProblemSolved(t, "456", problemID)
				env.assertProblemNotSolved(t, "789", problemID)
			}
			env.handleInteractionEvent(event)
			assert.Equal(t, 2, store.writes)
			assert.Len(t, env.Responder.edits, 2)
		})
	}
}

func TestWorkflow_ConfirmationCanRetryAfterFailureBeforeClaim(t *testing.T) {
	t.Parallel()
	for _, command := range []string{"problem", "ac"} {
		for _, failure := range []string{"permissions", "acknowledgement", "student resolution"} {
			if command == "problem" && failure == "student resolution" {
				continue
			}
			t.Run(command+"/"+failure, func(t *testing.T) {
				t.Parallel()
				env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
				allowed := true
				env.Discord.isMentorFunc = func(string, string) (bool, error) { return allowed, nil }
				classID := env.seedClass(t, "basicos", "Basicos")
				input := `[{"class":"basicos","name":"New","url":"https://example.com/new"}]`
				var problemID int64
				if command == "ac" {
					problemID = env.seedProblem(t, "Existing", "https://example.com/existing", classID, "easy", "Arrays")
					env.seedClassAccessRole(t, "basicos", "student")
					env.setRoles("123", []string{"student"})
					input = fmt.Sprintf(`[{"user_id":"123","problem_id":%d}]`, problemID)
				}
				preview := submitBatchPreview(t, env, command, input)
				switch failure {
				case "permissions":
					allowed = false
				case "acknowledgement":
					env.Responder.respondFunc = func(*discordgo.InteractionResponse) error { return errors.New("ack failed") }
				case "student resolution":
					env.Discord.getCachedMemberFunc = func(string, string) (*discordgo.Member, error) { return nil, errors.New("cache miss") }
					env.Discord.fetchMemberFunc = func(string, string) (*discordgo.Member, error) { return nil, errors.New("Discord unavailable") }
				}
				event := batchButtonEvent(t, preview, "Confirm")
				env.handleInteractionEvent(event)
				assert.Len(t, env.Responder.edits, 1, "failure before claim must preserve the preview")
				if failure != "acknowledgement" {
					require.Len(t, env.Responder.followups, 1)
					assert.Equal(t, discordgo.MessageFlagsEphemeral, env.Responder.followups[0].Flags)
				}
				problems, err := env.DB.GetProblemsByClass(classID)
				require.NoError(t, err)
				if command == "problem" {
					assert.Empty(t, problems)
				} else {
					env.assertProblemNotSolved(t, "123", problemID)
				}
				allowed = true
				env.Responder.respondFunc = nil
				env.Discord.getCachedMemberFunc = nil
				env.Discord.fetchMemberFunc = nil
				env.handleInteractionEvent(event)
				require.Len(t, env.Responder.edits, 2)
				assert.Contains(t, *env.Responder.edits[1].response.Content, "Batch completed")
				if command == "problem" {
					problems, err = env.DB.GetProblemsByClass(classID)
					require.NoError(t, err)
					assert.Len(t, problems, 1)
				} else {
					env.assertProblemSolved(t, "123", problemID)
				}
			})
		}
	}
}
