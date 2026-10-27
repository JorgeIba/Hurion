package workflow

import (
	"errors"
	"fmt"
	"testing"

	appbot "github.com/JorgeIba/Hurion/app/bot"
	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
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
	return submitBatchCommandPreview(t, env, command, "add_batch", input)
}

func submitBatchCommandPreview(t *testing.T, env *workflowEnv, command, subcommand, input string) *discordgo.WebhookEdit {
	t.Helper()
	member := &discordgo.Member{User: &discordgo.User{ID: "mentor"}}
	env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionApplicationCommand, GuildID: testGuildID, Member: member,
		Data: discordgo.ApplicationCommandInteractionData{Name: command, Options: []*discordgo.ApplicationCommandInteractionDataOption{{Name: subcommand, Type: discordgo.ApplicationCommandOptionSubCommand}}},
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
			env.setRoles("mentor", []string{"mentor_role"})
			classID := env.seedClass(t, "basicos", "Basicos")
			entry := `{"class":"basicos","name":"New","url":"https://example.com/new","topic":"Arrays"}`
			var problemID int64
			if test.command == "ac" {
				problemID = env.seedProblem(t, "Existing", "https://example.com/existing", classID, "easy", "Arrays")
				env.seedClassAccessRole(t, "basicos", "student")
				env.setRoles("123", []string{"student"})
				env.Discord.setDisplayName("123", "Preview Name")
				entry = fmt.Sprintf(`{"user_id":"123","problem_id":%d}`, problemID)
				if test.alreadyRecorded {
					require.NoError(t, env.Services.Solution.GrantACWithoutEnrollmentCheck(services.UserActor("mentor"), "123", problemID))
				}
			}
			input := "[" + entry + "]"
			if test.repeated {
				input = "[" + entry + "," + entry + "]"
			}
			preview := submitBatchPreview(t, env, test.command, input)
			if test.command == "ac" {
				env.Discord.setDisplayName("123", "Jorge Luis")
			}
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
				assert.NotContains(t, *receipt.Content, "Problem IDs")
				assert.Empty(t, receipt.Files)
			} else {
				env.assertProblemSolved(t, "123", problemID)
				user, err := env.DB.GetUserByUserID("123")
				require.NoError(t, err)
				assert.Equal(t, "Jorge Luis", user.DisplayName, "persist the confirmation name, including when the AC already exists")
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
			env.setRoles("mentor", []string{"mentor_role"})
			classID := env.seedClass(t, "basicos", "Basicos")
			otherClassID := env.seedClass(t, "other", "Other")
			input := `[{"class":"basicos","name":"Valid","url":"https://example.com/valid","topic":"Arrays"},{"class":"other","name":"Invalid after preview","url":"https://example.com/other","topic":"Arrays"}]`
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

type failProblemBatchRevalidationStore struct {
	data.DataStore
	classLookups int
	batchWrites  int
}

func (store *failProblemBatchRevalidationStore) GetClassesByClassCodes(classCodes []string) ([]data.Class, error) {
	store.classLookups++
	if store.classLookups == 2 {
		return nil, errors.New("database unavailable")
	}
	return store.DataStore.GetClassesByClassCodes(classCodes)
}

func (store *failProblemBatchRevalidationStore) AddProblemsBatch(problems []data.Problem) ([]int64, error) {
	store.batchWrites++
	return store.DataStore.AddProblemsBatch(problems)
}

func TestWorkflow_ProblemBatchRevalidationReadFailureReportsNoWrites(t *testing.T) {
	t.Parallel()
	env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
	env.setRoles("mentor", []string{"mentor_role"})
	classID := env.seedClass(t, "basicos", "Basicos")
	store := &failProblemBatchRevalidationStore{DataStore: env.DB}
	svcs := services.NewServices(store, env.authorization())
	responder := dispatcher.NewDiscordInteractionResponder(env.Responder)
	env.ProblemController = mentor.NewProblemInteractionController(svcs, responder)

	preview := submitBatchPreview(t, env, "problem", `[{"class":"basicos","name":"New","url":"https://example.com/new","topic":"Arrays"}]`)
	require.Equal(t, 1, store.classLookups, "preview preparation resolves the class once")
	event := batchButtonEvent(t, preview, "Confirm")
	env.handleInteractionEvent(event)

	require.Len(t, env.Responder.edits, 2)
	receipt := env.Responder.edits[1].response
	assert.Contains(t, *receipt.Content, "Could not validate the problem batch")
	assert.Contains(t, *receipt.Content, "No changes were applied")
	assert.NotContains(t, *receipt.Content, "outcome could not be confirmed")
	assert.Equal(t, 2, store.classLookups, "confirmation revalidation encounters the datastore failure")
	assert.Zero(t, store.batchWrites, "a failed revalidation must not reach the write method")

	problems, err := env.DB.GetProblemsByClass(classID)
	require.NoError(t, err)
	assert.Empty(t, problems)
}

// Lose acknowledgement after persistence to exercise uncertain storage outcomes.
type uncertainBatchStore struct {
	data.DataStore
	writes int
}

func (store *uncertainBatchStore) AddProblemsBatch(problems []data.Problem) ([]int64, error) {
	store.writes++
	problemIDs, err := store.DataStore.AddProblemsBatch(problems)
	if err == nil {
		return nil, errors.New("lost acknowledgement")
	}
	return problemIDs, err
}
func (store *uncertainBatchStore) AddACBatch(targets []data.ACKey, users []data.User) (int, error) {
	store.writes++
	count, err := store.DataStore.AddACBatch(targets, users)
	if err == nil {
		return 0, errors.New("lost acknowledgement")
	}
	return count, err
}

func TestWorkflow_BatchStorageFailureReportsProgressWithoutRetrying(t *testing.T) {
	t.Parallel()
	for _, command := range []string{"problem", "ac"} {
		t.Run(command, func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			env.setRoles("mentor", []string{"mentor_role"})
			classID := env.seedClass(t, "basicos", "Basicos")
			store := &uncertainBatchStore{DataStore: env.DB}
			svcs := services.NewServices(store, env.authorization())
			responder := dispatcher.NewDiscordInteractionResponder(env.Responder)
			env.ProblemController = mentor.NewProblemInteractionController(svcs, responder)
			env.ACController = mentor.NewACInteractionController(svcs, env.Discord, responder)
			input := `[{"class":"basicos","name":"One","url":"https://example.com/one","topic":"Arrays"},{"class":"basicos","name":"Two","url":"https://example.com/two","topic":"Arrays"},{"class":"basicos","name":"Three","url":"https://example.com/three","topic":"Arrays"}]`
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
			if command == "ac" {
				assert.Contains(t, *receipt.Content, "it may have been applied")
			} else {
				assert.Contains(t, *receipt.Content, "may have been applied")
			}
			assert.NotContains(t, *receipt.Content, "No changes were applied")
			assert.Empty(t, *receipt.Components)
			expectedWrites := 1
			assert.Equal(t, expectedWrites, store.writes, "do not retry writes")
			if command == "problem" {
				assert.Contains(t, *receipt.Content, "Confirmed problem creations: 0")
				problems, err := env.DB.GetProblemsByClass(classID)
				require.NoError(t, err)
				assert.Len(t, problems, 3)
			} else {
				assert.Contains(t, *receipt.Content, "Confirmed AC registrations: 0")
				env.assertProblemSolved(t, "123", problemID)
				env.assertProblemSolved(t, "456", problemID)
				env.assertProblemSolved(t, "789", problemID)
			}
			env.handleInteractionEvent(event)
			assert.Equal(t, expectedWrites, store.writes)
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
				env.setRoles("mentor", []string{"mentor_role"})
				classID := env.seedClass(t, "basicos", "Basicos")
				input := `[{"class":"basicos","name":"New","url":"https://example.com/new","topic":"Arrays"}]`
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
					env.setRoles("mentor", nil)
				case "acknowledgement":
					env.Responder.respondFunc = func(*discordgo.InteractionResponse) error { return errors.New("ack failed") }
				case "student resolution":
					env.Discord.getCachedMemberFunc = func(_, userID string) (*discordgo.Member, error) {
						if userID == "mentor" {
							return &discordgo.Member{User: &discordgo.User{ID: userID}, Roles: env.Discord.userRoles[userID]}, nil
						}
						return nil, errors.New("cache miss")
					}
					env.Discord.fetchMemberFunc = func(guildID, userID string) (*discordgo.Member, error) {
						if userID == "mentor" {
							return env.Discord.GetCachedMember(guildID, userID)
						}
						return nil, errors.New("Discord unavailable")
					}
				}
				event := batchButtonEvent(t, preview, "Confirm")
				env.handleInteractionEvent(event)
				if failure == "permissions" {
					require.Len(t, env.Responder.edits, 2, "service authorization fails after claiming confirmation")
					assert.Contains(t, *env.Responder.edits[1].response.Content, "No changes were applied")
					problems, err := env.DB.GetProblemsByClass(classID)
					require.NoError(t, err)
					if command == "problem" {
						assert.Empty(t, problems)
					} else {
						env.assertProblemNotSolved(t, "123", problemID)
					}
					env.setRoles("mentor", []string{"mentor_role"})
					env.handleInteractionEvent(event)
					assert.Len(t, env.Responder.edits, 2, "claimed preview cannot execute again")
					require.NotEmpty(t, env.Responder.responses)
					assert.Contains(t, env.Responder.responses[len(env.Responder.responses)-1].response.Data.Content, "available")
					newPreview := submitBatchPreview(t, env, command, input)
					env.handleInteractionEvent(batchButtonEvent(t, newPreview, "Confirm"))
					assert.Contains(t, *env.Responder.edits[len(env.Responder.edits)-1].response.Content, "Batch completed")
					return
				}
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
				env.setRoles("mentor", []string{"mentor_role"})
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

func TestWorkflow_ACBatchRejectsMovedProblemBeforeAnyWrite(t *testing.T) {
	t.Parallel()
	env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
	env.setRoles("mentor", []string{"mentor_role"})
	original := env.seedClass(t, "original", "Original")
	destination := env.seedClass(t, "destination", "Destination")
	first := env.seedProblem(t, "First", "https://example.com/first", original, "easy", "Arrays")
	second := env.seedProblem(t, "Second", "https://example.com/second", original, "easy", "Arrays")
	env.seedClassAccessRole(t, "original", "student")
	env.seedClassAccessRole(t, "destination", "student")
	env.setRoles("123", []string{"student"})
	input := fmt.Sprintf(`[{"user_id":"123","problem_id":%d},{"user_id":"123","problem_id":%d}]`, first, second)
	preview := submitBatchPreview(t, env, "ac", input)
	changed, err := env.DB.GetProblemByID(second)
	require.NoError(t, err)
	changed.ClassID = destination
	require.NoError(t, env.DB.ReplaceProblemByProblemID(second, changed))
	env.handleInteractionEvent(batchButtonEvent(t, preview, "Confirm"))
	require.Len(t, env.Responder.edits, 2)
	assert.Contains(t, *env.Responder.edits[1].response.Content, "Entry 2")
	assert.Contains(t, *env.Responder.edits[1].response.Content, "No changes were applied")
	env.assertProblemNotSolved(t, "123", first)
	env.assertProblemNotSolved(t, "123", second)
	_, err = env.DB.GetUserByUserID("123")
	assert.Error(t, err, "invalidated batches must not create students")
}

func TestWorkflow_ACBatchUsesCurrentRecordedStateAtConfirmation(t *testing.T) {
	t.Parallel()
	for _, initiallyRecorded := range []bool{false, true} {
		t.Run(fmt.Sprintf("initially recorded=%t", initiallyRecorded), func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			env.setRoles("mentor", []string{"mentor_role"})
			classID := env.seedClass(t, "basicos", "Basicos")
			id := env.seedProblem(t, "Existing", "https://example.com/existing", classID, "easy", "Arrays")
			env.seedClassAccessRole(t, "basicos", "student")
			env.setRoles("123", []string{"student"})
			if initiallyRecorded {
				require.NoError(t, env.Services.Solution.GrantACWithoutEnrollmentCheck(services.UserActor("mentor"), "123", id))
			}
			preview := submitBatchPreview(t, env, "ac", fmt.Sprintf(`[{"user_id":"123","problem_id":%d}]`, id))
			recorded, already := 0, 1
			if initiallyRecorded {
				require.NoError(t, env.DB.RemoveAC("123", id))
				recorded, already = 1, 0
			} else {
				require.NoError(t, env.Services.Solution.GrantACWithoutEnrollmentCheck(services.UserActor("mentor"), "123", id))
			}
			env.handleInteractionEvent(batchButtonEvent(t, preview, "Confirm"))
			require.Len(t, env.Responder.edits, 2)
			assert.Contains(t, *env.Responder.edits[1].response.Content, fmt.Sprintf("ACs recorded: %d. Already recorded: %d.", recorded, already))
			env.assertProblemSolved(t, "123", id)
		})
	}
}

type failingACValidationStore struct {
	data.DataStore
	failReads bool
}

func (store *failingACValidationStore) GetACValidationData(targets []data.ACKey) (map[data.ACKey]data.ACValidationData, error) {
	if store.failReads {
		return nil, errors.New("validation read unavailable")
	}
	return store.DataStore.GetACValidationData(targets)
}

func TestWorkflow_ACBatchReadFailureReportsNoWrites(t *testing.T) {
	t.Parallel()
	for _, phase := range []string{"preview", "confirmation"} {
		t.Run(phase, func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			env.setRoles("mentor", []string{"mentor_role"})
			classID := env.seedClass(t, "basicos", "Basicos")
			id := env.seedProblem(t, "Existing", "https://example.com/existing", classID, "easy", "Arrays")
			env.seedClassAccessRole(t, "basicos", "student")
			env.setRoles("123", []string{"student"})
			store := &failingACValidationStore{DataStore: env.DB, failReads: phase == "preview"}
			env.ACController = mentor.NewACInteractionController(services.NewServices(store, env.authorization()), env.Discord, dispatcher.NewDiscordInteractionResponder(env.Responder))
			input := fmt.Sprintf(`[{"user_id":"123","problem_id":%d}]`, id)
			if phase == "preview" {
				env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
					Type: discordgo.InteractionModalSubmit, GuildID: testGuildID, Member: &discordgo.Member{User: &discordgo.User{ID: "mentor"}},
					Data: discordgo.ModalSubmitInteractionData{CustomID: string(discordcommands.ACBatch), Components: []discordgo.MessageComponent{
						&discordgo.Label{Component: &discordgo.TextInput{CustomID: "json", Value: input}},
						&discordgo.Label{Component: &discordgo.FileUpload{CustomID: "file"}},
					}},
				}})
			} else {
				preview := submitBatchPreview(t, env, "ac", input)
				store.failReads = true
				event := batchButtonEvent(t, preview, "Confirm")
				env.handleInteractionEvent(event)
				env.handleInteractionEvent(event)
				require.Len(t, env.Responder.edits, 2, "a consumed batch is not retried after a read failure")
			}
			require.NotEmpty(t, env.Responder.edits)
			receipt := env.Responder.edits[len(env.Responder.edits)-1].response
			require.NotNil(t, receipt.Content)
			assert.Contains(t, *receipt.Content, "Could not validate")
			assert.Contains(t, *receipt.Content, "No changes were applied")
			assert.NotContains(t, *receipt.Content, "may also have been applied")
			if receipt.Components != nil {
				assert.Empty(t, *receipt.Components)
			}
			env.assertProblemNotSolved(t, "123", id)
			_, err := env.DB.GetUserByUserID("123")
			assert.Error(t, err, "failed validation reads cannot create students")
		})
	}
}
