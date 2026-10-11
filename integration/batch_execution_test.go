package integration

import (
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"testing"

	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_ExecuteProblemBatchCreatesOrderedRepeatedProblems(t *testing.T) {
	t.Parallel()
	env := setupIntegrationEnv(t)
	firstClass := env.SeedClass(t, "basicos", "Basics")
	secondClass := env.SeedClass(t, "avanzados", "Advanced")
	repeated := service_model.CreateProblemParams{ClassCode: "basicos", Name: "Repeated", URL: "https://example.com/repeated"}
	prepared, issues := env.Services.Problem.PrepareCreationBatch([]service_model.CreateProblemParams{
		repeated, {ClassCode: "avanzados", Name: "Other", URL: "https://example.com/other"}, repeated,
	})
	require.Empty(t, issues)
	result, issues, err := env.Services.Problem.ExecuteCreationBatch(prepared)
	require.NoError(t, err)
	require.Empty(t, issues)
	require.Len(t, result.CreatedProblemIDs, 3)
	assert.Less(t, result.CreatedProblemIDs[0], result.CreatedProblemIDs[1])
	assert.Less(t, result.CreatedProblemIDs[1], result.CreatedProblemIDs[2])
	for index, classID := range []int64{firstClass, secondClass, firstClass} {
		problem, err := env.DB.GetProblemByID(result.CreatedProblemIDs[index])
		require.NoError(t, err)
		assert.Equal(t, classID, problem.ClassID)
		assert.Equal(t, prepared[index].Params.Name, problem.Name)
	}
}

func TestIntegration_ExecuteProblemBatchValidatesEveryTargetBeforeWrites(t *testing.T) {
	t.Parallel()
	env := setupIntegrationEnv(t)
	validClass := env.SeedClass(t, "basicos", "Basics")
	removedClass := env.SeedClass(t, "removed", "Removed")
	prepared, issues := env.Services.Problem.PrepareCreationBatch([]service_model.CreateProblemParams{
		{ClassCode: "basicos", Name: "Valid", URL: "https://example.com/valid"},
		{ClassCode: "removed", Name: "Gone", URL: "https://example.com/gone"},
	})
	require.Empty(t, issues)
	require.NoError(t, env.DB.DeleteClass(removedClass))
	result, issues, err := env.Services.Problem.ExecuteCreationBatch(prepared)
	require.NoError(t, err)
	assert.Empty(t, result.CreatedProblemIDs)
	require.Len(t, issues, 1)
	assert.Equal(t, 2, issues[0].Entry)
	problems, err := env.DB.GetProblemsByClass(validClass)
	require.NoError(t, err)
	assert.Empty(t, problems, "a valid first entry must not be written when a later entry fails validation")
}

func TestIntegration_ExecuteProblemPatchBatchUpdatesOnlySpecifiedFields(t *testing.T) {
	t.Parallel()
	env := setupIntegrationEnv(t)
	sourceClassID := env.SeedClass(t, "basicos", "Basics")
	destinationClassID := env.SeedClass(t, "avanzados", "Advanced")
	difficulty, topic, priority := "easy", "Arrays", "recommended"
	problemID, err := env.DB.AddProblem(data.Problem{
		Name: "Two Sum", URL: "https://example.com/two-sum", ClassID: sourceClassID,
		Difficulty: &difficulty, Topic: &topic, Priority: &priority,
	})
	require.NoError(t, err)
	require.NoError(t, env.DB.EnsureUser(data.User{UserID: "123", DisplayName: "Ana"}))
	require.NoError(t, env.DB.AddAC("123", problemID))
	original, err := env.DB.GetProblemByID(problemID)
	require.NoError(t, err)

	var inputs []service_model.PatchProblemParams
	input := fmt.Sprintf(`[{"problem_id":%d,"class":"avanzados","difficulty":"hard","topic":null}]`, problemID)
	require.NoError(t, jsonv2.Unmarshal([]byte(input), &inputs))
	require.Len(t, inputs, 1)
	assert.False(t, inputs[0].Name.Set)
	assert.False(t, inputs[0].URL.Set)
	assert.False(t, inputs[0].Priority.Set)
	assert.True(t, inputs[0].Topic.Set)
	assert.Nil(t, inputs[0].Topic.Value)

	// A serialized patch must retain the same update and clear instructions.
	encoded, err := jsonv2.Marshal(inputs)
	require.NoError(t, err)
	assert.JSONEq(t, input, string(encoded))
	var decodedInputs []service_model.PatchProblemParams
	require.NoError(t, jsonv2.Unmarshal(encoded, &decodedInputs))
	assert.Equal(t, inputs, decodedInputs)
	inputs = decodedInputs

	prepared, issues := env.Services.Problem.PrepareProblemPatchBatch(inputs)
	require.Empty(t, issues)
	require.Len(t, prepared, 1)
	result, issues, err := env.Services.Problem.ExecuteProblemPatchBatch(prepared)
	require.NoError(t, err)
	require.Empty(t, issues)
	assert.Equal(t, []int64{problemID}, result.PatchedProblemIDs)

	updated, err := env.DB.GetProblemByID(problemID)
	require.NoError(t, err)
	assert.Equal(t, original.Name, updated.Name)
	assert.Equal(t, original.URL, updated.URL)
	assert.Equal(t, destinationClassID, updated.ClassID)
	require.NotNil(t, updated.Difficulty)
	assert.Equal(t, "hard", *updated.Difficulty)
	assert.Nil(t, updated.Topic)
	require.NotNil(t, updated.Priority)
	assert.Equal(t, priority, *updated.Priority)
	assert.Equal(t, original.CreatedAt, updated.CreatedAt)

	solved, err := env.DB.HasUserSolvedProblem("123", problemID)
	require.NoError(t, err)
	assert.True(t, solved)
}

func TestIntegration_ExecuteACBatchRechecksSolvesAndCountsRepeatedEntries(t *testing.T) {
	t.Parallel()
	for _, scenario := range []string{"new repeated solve", "solve added after preview", "solve removed after preview"} {
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			env := setupIntegrationEnv(t)
			classID := env.SeedClass(t, "basicos", "Basics")
			problemID := env.SeedProblem(t, "Problem", "https://example.com", classID, "easy", "Arrays")
			env.SeedClassAccessRole(t, "basicos", "student")
			students := map[string]service_model.Principal{"123": {UserID: "123", Roles: []string{"student"}}}
			if scenario == "solve removed after preview" {
				require.NoError(t, env.DB.EnsureUser(data.User{UserID: "123", DisplayName: "Ana"}))
				require.NoError(t, env.DB.AddAC("123", problemID))
			}
			input := service_model.CreateACParams{UserID: "123", ProblemID: problemID}
			prepared, issues, err := env.Services.Solution.PrepareCreationBatch([]service_model.CreateACParams{input, input}, students)
			require.NoError(t, err)
			require.Empty(t, issues)
			if scenario == "solve added after preview" {
				require.NoError(t, env.DB.EnsureUser(data.User{UserID: "123", DisplayName: "Ana"}))
				require.NoError(t, env.DB.AddAC("123", problemID))
			} else if scenario == "solve removed after preview" {
				require.NoError(t, env.DB.RemoveAC("123", problemID))
			}
			result, issues, err := env.Services.Solution.ExecuteCreationBatch(prepared, students)
			require.NoError(t, err)
			require.Empty(t, issues)
			if scenario == "solve added after preview" {
				assert.Equal(t, service_model.ACBatchResult{AlreadyRecorded: 2}, result)
			} else {
				assert.Equal(t, service_model.ACBatchResult{Recorded: 1, AlreadyRecorded: 1}, result)
			}
			solved, err := env.DB.HasUserSolvedProblem("123", problemID)
			require.NoError(t, err)
			assert.True(t, solved)
		})
	}
}

func TestIntegration_ExecuteACBatchRevalidatesAllStudentsAndTargetsBeforeWrites(t *testing.T) {
	t.Parallel()
	for _, scenario := range []string{"role revoked", "student unresolved", "access role removed", "problem deleted", "problem moved"} {
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			env := setupIntegrationEnv(t)
			classID := env.SeedClass(t, "basicos", "Basics")
			firstProblem := env.SeedProblem(t, "First", "https://example.com/first", classID, "easy", "Arrays")
			secondProblem := env.SeedProblem(t, "Second", "https://example.com/second", classID, "easy", "Arrays")
			env.SeedClassAccessRole(t, "basicos", "student")
			students := map[string]service_model.Principal{
				"123": {UserID: "123", Roles: []string{"student"}},
				"456": {UserID: "456", Roles: []string{"student"}},
			}
			prepared, issues, err := env.Services.Solution.PrepareCreationBatch([]service_model.CreateACParams{
				{UserID: "123", ProblemID: firstProblem}, {UserID: "456", ProblemID: secondProblem},
			}, students)
			require.NoError(t, err)
			require.Empty(t, issues)
			switch scenario {
			case "role revoked":
				students["456"] = service_model.Principal{UserID: "456", Roles: []string{"other"}}
			case "student unresolved":
				delete(students, "456")
			case "access role removed":
				require.NoError(t, env.DB.RemoveClassAccessRole(data.ClassAccessRole{ClassCode: "basicos", RoleID: "student"}))
			case "problem deleted":
				_, err := env.DB.RemoveProblemByID(secondProblem)
				require.NoError(t, err)
			case "problem moved":
				otherClass := env.SeedClass(t, "avanzados", "Advanced")
				env.SeedClassAccessRole(t, "avanzados", "student")
				problem, err := env.DB.GetProblemByID(secondProblem)
				require.NoError(t, err)
				problem.ClassID = otherClass
				require.NoError(t, env.DB.ReplaceProblemByProblemID(secondProblem, problem))
			}
			result, issues, err := env.Services.Solution.ExecuteCreationBatch(prepared, students)
			require.NoError(t, err)
			assert.Zero(t, result.Recorded)
			assert.Zero(t, result.AlreadyRecorded)
			require.NotEmpty(t, issues)
			if scenario != "access role removed" {
				require.Len(t, issues, 1)
				assert.Equal(t, 2, issues[0].Entry)
			}
			for _, userID := range []string{"123", "456"} {
				_, err := env.DB.GetUserByUserID(userID)
				assert.Error(t, err, "validation failure must not create student records")
			}
			solved, err := env.DB.HasUserSolvedProblem("123", firstProblem)
			require.NoError(t, err)
			assert.False(t, solved, "no earlier valid entry may be written")
		})
	}
}

// Inject storage failures to check partial problem writes and atomic AC writes.
type failingBatchStore struct {
	data.DataStore
	failure                 error
	problemWrites, acWrites int
}

func (store *failingBatchStore) AddProblem(problem data.Problem) (int64, error) {
	store.problemWrites++
	if store.problemWrites == 2 {
		return 0, store.failure
	}
	return store.DataStore.AddProblem(problem)
}

func (store *failingBatchStore) AddACBatch(targets []data.ACKey) (int, error) {
	store.acWrites++
	// Force a foreign-key failure after valid pairs reach the bulk insert.
	withInvalidTarget := append([]data.ACKey(nil), targets...)
	withInvalidTarget = append(withInvalidTarget, data.ACKey{UserID: "456", ProblemID: -1})
	count, err := store.DataStore.AddACBatch(withInvalidTarget)
	return count, errors.Join(store.failure, err)
}

func TestIntegration_ExecuteBatchStopsOnStorageFailureAndReportsAcknowledgedWrites(t *testing.T) {
	t.Parallel()
	for _, family := range []string{"problem", "ac"} {
		t.Run(family, func(t *testing.T) {
			t.Parallel()
			env := setupIntegrationEnv(t)
			classID := env.SeedClass(t, "basicos", "Basics")
			failure := errors.New("storage unavailable")
			store := &failingBatchStore{DataStore: env.DB, failure: failure}
			svcs := services.NewServices(store)
			if family == "problem" {
				input := service_model.CreateProblemParams{ClassCode: "basicos", Name: "Repeated", URL: "https://example.com"}
				prepared, issues := svcs.Problem.PrepareCreationBatch([]service_model.CreateProblemParams{input, input, input})
				require.Empty(t, issues)
				result, issues, err := svcs.Problem.ExecuteCreationBatch(prepared)
				require.ErrorIs(t, err, failure)
				assert.Contains(t, err.Error(), "entry 2")
				assert.Empty(t, issues)
				require.Len(t, result.CreatedProblemIDs, 1)
				assert.Equal(t, 2, store.problemWrites, "must not attempt the third operation")
				problems, err := env.DB.GetProblemsByClass(classID)
				require.NoError(t, err)
				require.Len(t, problems, 1)
				assert.Equal(t, result.CreatedProblemIDs[0], problems[0].ID)
			} else {
				problemID := env.SeedProblem(t, "Problem", "https://example.com", classID, "easy", "Arrays")
				env.SeedClassAccessRole(t, "basicos", "student")
				students := map[string]service_model.Principal{}
				var inputs []service_model.CreateACParams
				for _, userID := range []string{"123", "456", "789"} {
					students[userID] = service_model.Principal{UserID: userID, Roles: []string{"student"}}
					inputs = append(inputs, service_model.CreateACParams{UserID: userID, ProblemID: problemID})
				}
				prepared, issues, err := svcs.Solution.PrepareCreationBatch(inputs, students)
				require.NoError(t, err)
				require.Empty(t, issues)
				result, issues, err := svcs.Solution.ExecuteCreationBatch(prepared, students)
				require.ErrorIs(t, err, failure)
				assert.Contains(t, err.Error(), "AC batch")
				assert.Empty(t, issues)
				assert.Zero(t, result)
				assert.Equal(t, 1, store.acWrites)
				solved, err := env.DB.HasUserSolvedProblem("123", problemID)
				require.NoError(t, err)
				assert.False(t, solved, "the complete AC batch must roll back")
				solved, err = env.DB.HasUserSolvedProblem("456", problemID)
				require.NoError(t, err)
				assert.False(t, solved)
				_, err = env.DB.GetUserByUserID("456")
				assert.Error(t, err, "user creation must roll back with the solves")
				_, err = env.DB.GetUserByUserID("789")
				assert.Error(t, err, "unattempted operations must not create user records")
			}
		})
	}
}
