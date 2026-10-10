package workflow

import (
	"errors"
	"fmt"
	"testing"

	appbot "github.com/JorgeIba/Hurion/app/bot"
	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/JorgeIba/Hurion/app/controllers/mentor"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflow_EditBatchPreservesIdentityAndSolutionsAndReplacesFields(t *testing.T) {
	t.Parallel()
	for _, repeated := range []bool{false, true} {
		t.Run(fmt.Sprintf("repeated target=%t", repeated), func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			env.Discord.isMentorFunc = func(string, string) (bool, error) { return true, nil }
			source := env.seedClass(t, "source", "Source")
			destination := env.seedClass(t, "destination", "Destination")
			id, err := env.DB.AddProblem(data.Problem{Name: "Original", URL: "https://example.com/original", ClassID: source, Difficulty: new("hard"), Topic: new("Trees"), Priority: new("recommended")})
			require.NoError(t, err)
			env.seedUser(t, "student", "Student")
			require.NoError(t, env.DB.AddAC("student", id))
			before, err := env.DB.GetProblemByID(id)
			require.NoError(t, err)
			solvedBefore, err := env.DB.GetSolvedProblemsByUserAndClass("student", source)
			require.NoError(t, err)
			entry := fmt.Sprintf(`{"problem_id":%d,"class":"destination","name":" Updated @everyone ","url":" https://example.com/updated "}`, id)
			input := "[" + entry + "]"
			if repeated {
				input = "[" + entry + "," + fmt.Sprintf(`{"problem_id":%d,"class":"destination","name":"Last","url":"https://example.com/last"}`, id) + "]"
			}
			preview := submitBatchCommandPreview(t, env, "problem", "edit_batch", input)
			untouched, err := env.DB.GetProblemByID(id)
			require.NoError(t, err)
			assert.Equal(t, before, untouched, "preview must not write")
			assert.Contains(t, *preview.Content, "Destination classes: 1")
			event := batchButtonEvent(t, preview, "Confirm")
			env.handleInteractionEvent(event)
			after, err := env.DB.GetProblemByID(id)
			require.NoError(t, err)
			assert.Equal(t, before.ID, after.ID)
			assert.Equal(t, before.CreatedAt, after.CreatedAt)
			assert.Equal(t, destination, after.ClassID)
			assert.Nil(t, after.Difficulty)
			assert.Nil(t, after.Topic)
			assert.Nil(t, after.Priority)
			expectedName, expectedURL, count := "Updated @everyone", "https://example.com/updated", 1
			if repeated {
				expectedName, expectedURL, count = "Last", "https://example.com/last", 2
			}
			assert.Equal(t, expectedName, after.Name)
			assert.Equal(t, expectedURL, after.URL)
			solvedAfter, err := env.DB.GetSolvedProblemsByUserAndClass("student", destination)
			require.NoError(t, err)
			assert.Equal(t, solvedBefore, solvedAfter)
			require.Len(t, env.Responder.edits, 2)
			receipt := env.Responder.edits[1].response
			assert.Contains(t, *receipt.Content, fmt.Sprintf("Problem updates applied: %d", count))
			assert.Contains(t, *receipt.Content, fmt.Sprint(id))
			assert.Empty(t, *receipt.Components)
			assert.Empty(t, receipt.AllowedMentions.Parse)
			env.handleInteractionEvent(event)
			assert.Len(t, env.Responder.edits, 2, "consumed edits cannot be repeated")
		})
	}
}

func submitProblemEditBatchInput(env *workflowEnv, input string) {
	env.handleInteractionEvent(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionModalSubmit, GuildID: testGuildID, Member: &discordgo.Member{User: &discordgo.User{ID: "mentor"}},
		Data: discordgo.ModalSubmitInteractionData{CustomID: string(discordcommands.ProblemEditBatch), Components: []discordgo.MessageComponent{
			&discordgo.Label{Component: &discordgo.TextInput{CustomID: "json", Value: input}},
			&discordgo.Label{Component: &discordgo.FileUpload{CustomID: "file"}},
		}},
	}})
}

func TestWorkflow_EditBatchInvalidInputDoesNotPartiallyUpdate(t *testing.T) {
	t.Parallel()
	for _, invalid := range []string{
		`{"problem_id":1,"class":"source","name":"Bad","url":"https://example.com","typo":true}`,
		`{"problem_id":"1","class":"source","name":"Bad","url":"https://example.com"}`,
		`{"class":"source","name":"Missing ID","url":"https://example.com"}`,
		`{"problem_id":-1,"class":"source","name":"Bad","url":"https://example.com"}`,
		`{"problem_id":9223372036854775807,"class":"source","name":"Missing","url":"https://example.com"}`,
		`{"problem_id":1,"class":"missing","name":"Bad","url":"https://example.com"}`,
	} {
		t.Run(invalid, func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			env.Discord.isMentorFunc = func(string, string) (bool, error) { return true, nil }
			source := env.seedClass(t, "source", "Source")
			id := env.seedProblem(t, "Original", "https://example.com/original", source, "hard", "Trees")
			before, err := env.DB.GetProblemByID(id)
			require.NoError(t, err)
			input := fmt.Sprintf(`[{"problem_id":%d,"class":"source","name":"Valid","url":"https://example.com/new"},%s]`, id, invalid)
			submitProblemEditBatchInput(env, input)
			require.Len(t, env.Responder.edits, 1)
			receipt := env.Responder.edits[0].response
			assert.Contains(t, *receipt.Content, "Entry 2")
			if receipt.Components != nil {
				assert.Empty(t, *receipt.Components)
			}
			after, err := env.DB.GetProblemByID(id)
			require.NoError(t, err)
			assert.Equal(t, before, after)
		})
	}
}

func TestWorkflow_EditBatchRevalidatesAllTargetsBeforeWriting(t *testing.T) {
	t.Parallel()
	for _, change := range []string{"deleted target", "reused destination"} {
		t.Run(change, func(t *testing.T) {
			t.Parallel()
			env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
			env.Discord.isMentorFunc = func(string, string) (bool, error) { return true, nil }
			source := env.seedClass(t, "source", "Source")
			destination := env.seedClass(t, "destination", "Destination")
			first := env.seedProblem(t, "First", "https://example.com/first", source, "easy", "Arrays")
			second := env.seedProblem(t, "Second", "https://example.com/second", source, "easy", "Arrays")
			before, err := env.DB.GetProblemByID(first)
			require.NoError(t, err)
			input := fmt.Sprintf(`[{"problem_id":%d,"class":"destination","name":"New first","url":"https://example.com/new1"},{"problem_id":%d,"class":"destination","name":"New second","url":"https://example.com/new2"}]`, first, second)
			preview := submitBatchCommandPreview(t, env, "problem", "edit_batch", input)
			if change == "deleted target" {
				_, err = env.DB.RemoveProblemByID(second)
				require.NoError(t, err)
			} else {
				require.NoError(t, env.DB.DeleteClass(destination))
				env.seedClass(t, "destination", "Replacement")
			}
			env.handleInteractionEvent(batchButtonEvent(t, preview, "Confirm"))
			after, err := env.DB.GetProblemByID(first)
			require.NoError(t, err)
			assert.Equal(t, before, after)
			require.Len(t, env.Responder.edits, 2)
			assert.Contains(t, *env.Responder.edits[1].response.Content, "No changes were applied")
			assert.Empty(t, *env.Responder.edits[1].response.Components)
		})
	}
}

func TestWorkflow_EditBatchCancelAndForgedCreateButtonDoNotWrite(t *testing.T) {
	t.Parallel()
	env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
	env.Discord.isMentorFunc = func(string, string) (bool, error) { return true, nil }
	class := env.seedClass(t, "source", "Source")
	id := env.seedProblem(t, "Original", "https://example.com/original", class, "easy", "Arrays")
	before, err := env.DB.GetProblemByID(id)
	require.NoError(t, err)
	preview := submitBatchCommandPreview(t, env, "problem", "edit_batch", fmt.Sprintf(`[{"problem_id":%d,"class":"source","name":"Changed","url":"https://example.com/new"}]`, id))
	forged := batchButtonEvent(t, preview, "Confirm")
	button := forged.MessageComponentData()
	batchID, err := interactiontransport.ParseProblemEditBatchConfirmID(button.CustomID)
	require.NoError(t, err)
	button.CustomID = interactiontransport.ProblemBatchConfirmID(batchID)
	forged.Data = button
	env.handleInteractionEvent(forged)
	require.Len(t, env.Responder.edits, 1, "create button cannot consume edit cache")
	assert.Equal(t, discord.BatchUnavailableMessage, env.Responder.responses[len(env.Responder.responses)-1].response.Data.Content)
	env.handleInteractionEvent(batchButtonEvent(t, preview, "Cancel"))
	require.Len(t, env.Responder.edits, 2)
	assert.Contains(t, *env.Responder.edits[1].response.Content, "Batch cancelled")
	env.handleInteractionEvent(batchButtonEvent(t, preview, "Confirm"))
	assert.Len(t, env.Responder.edits, 2)
	after, err := env.DB.GetProblemByID(id)
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

type uncertainProblemUpdateStore struct {
	data.DataStore
	writes int
}

func (store *uncertainProblemUpdateStore) UpdateProblemByProblemID(id int64, problem data.Problem) error {
	store.writes++
	err := store.DataStore.UpdateProblemByProblemID(id, problem)
	if err == nil && store.writes == 2 {
		return errors.New("lost acknowledgement")
	}
	return err
}
func TestWorkflow_EditBatchStorageFailureStopsLaterWritesAndReportsProgress(t *testing.T) {
	t.Parallel()
	env := setupWorkflowEnv(t, appbot.BotConfig{GuildID: testGuildID})
	env.Discord.isMentorFunc = func(string, string) (bool, error) { return true, nil }
	class := env.seedClass(t, "source", "Source")
	ids := []int64{env.seedProblem(t, "One", "https://example.com/one", class, "easy", "Arrays"), env.seedProblem(t, "Two", "https://example.com/two", class, "easy", "Arrays"), env.seedProblem(t, "Three", "https://example.com/three", class, "easy", "Arrays")}
	store := &uncertainProblemUpdateStore{DataStore: env.DB}
	env.ProblemController = mentor.NewProblemInteractionController(services.NewServices(store), env.Discord, dispatcher.NewDiscordInteractionResponder(env.Responder))
	input := fmt.Sprintf(`[{"problem_id":%d,"class":"source","name":"Changed 1","url":"https://example.com/new1"},{"problem_id":%d,"class":"source","name":"Changed 2","url":"https://example.com/new2"},{"problem_id":%d,"class":"source","name":"Changed 3","url":"https://example.com/new3"}]`, ids[0], ids[1], ids[2])
	preview := submitBatchCommandPreview(t, env, "problem", "edit_batch", input)
	event := batchButtonEvent(t, preview, "Confirm")
	env.handleInteractionEvent(event)
	assert.Equal(t, 2, store.writes)
	require.Len(t, env.Responder.edits, 2)
	receipt := env.Responder.edits[1].response
	assert.Contains(t, *receipt.Content, "Confirmed problem updates: 1")
	assert.Contains(t, *receipt.Content, "may also have been applied")
	third, err := env.DB.GetProblemByID(ids[2])
	require.NoError(t, err)
	assert.Equal(t, "Three", third.Name)
	env.handleInteractionEvent(event)
	assert.Equal(t, 2, store.writes)
	assert.Len(t, env.Responder.edits, 2)
}
