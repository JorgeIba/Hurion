package services

import (
	"errors"
	"fmt"
	"testing"

	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestPrepareProblemBatchNormalizesAndKeepsRepeatedProblems(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	store.EXPECT().GetClassesByClassCodes([]string{"basicos"}).Return([]data.Class{{ID: 10, ClassCode: "basicos", Name: "Basics"}}, nil)
	difficulty := " HARD "
	input := service_model.CreateProblemParams{ClassCode: " basicos ", Name: " Two Sum ", URL: " https://example.com/problem ", Difficulty: &difficulty}
	prepared, issues := NewServices(store).Batch.PrepareProblemBatch([]service_model.CreateProblemParams{input, input})
	assert.Empty(t, issues)
	if assert.Len(t, prepared, 2) {
		assert.Equal(t, 1, prepared[0].Entry)
		assert.Equal(t, 2, prepared[1].Entry)
		assert.Equal(t, int64(10), prepared[0].Class.ID)
		assert.Equal(t, "basicos", prepared[0].Params.ClassCode)
		assert.Equal(t, "Two Sum", prepared[0].Params.Name)
		assert.Equal(t, "https://example.com/problem", prepared[0].Params.URL)
		assert.Equal(t, "hard", *prepared[0].Params.Difficulty)
		assert.Equal(t, prepared[0].Params, prepared[1].Params)
	}
	assert.Equal(t, " HARD ", difficulty)
	assert.Equal(t, " basicos ", input.ClassCode)
}

func TestPrepareProblemBatchReportsEveryInvalidEntry(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	store.EXPECT().GetClassesByClassCodes([]string{"missing"}).Return(nil, nil)
	input := []service_model.CreateProblemParams{
		{},
		{ClassCode: "missing", Name: "Valid", URL: "https://example.com/problem"},
		{Name: "Other", ClassCode: "basicos", URL: "javascript:alert(1)"},
	}
	prepared, issues := NewServices(store).Batch.PrepareProblemBatch(input)
	assert.Nil(t, prepared)
	if assert.Len(t, issues, 3) {
		assert.Equal(t, 1, issues[0].Entry)
		assert.IsType(t, &serviceerr.ValidationError{}, issues[0].Err)
		assert.Contains(t, issues[0].Err.Error(), "problem name is required")
		assert.Contains(t, issues[0].Err.Error(), "class code is required")
		assert.Equal(t, 2, issues[1].Entry)
		assert.IsType(t, &serviceerr.ClassCodeNotFoundError{}, issues[1].Err)
		assert.Equal(t, 3, issues[2].Entry)
	}
}

func TestPrepareACBatchUsesStudentAndClassWithoutWrites(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	store.EXPECT().GetACValidationData([]data.ACKey{{UserID: "123", ProblemID: 42}}).Return(map[data.ACKey]data.ACValidationData{{UserID: "123", ProblemID: 42}: {Problem: &data.Problem{ID: 42, ClassID: 10, Name: "Two Sum"}, Class: &data.Class{ID: 10, ClassCode: "basicos"}, AccessRoleIDs: []string{"student"}}}, nil)

	prepared, issues, err := NewServices(store).Batch.PrepareACBatch([]service_model.CreateACParams{{UserID: " 00123 ", ProblemID: 42}}, map[string]service_model.Principal{"123": {UserID: "123", DisplayName: "Ana", Roles: []string{"student"}}})
	require.NoError(t, err)
	assert.Empty(t, issues)
	if assert.Len(t, prepared, 1) {
		assert.Equal(t, 1, prepared[0].Entry)
		assert.Equal(t, "123", prepared[0].User.UserID)
		assert.Equal(t, "Ana", prepared[0].User.DisplayName)
		assert.Equal(t, int64(42), prepared[0].Problem.ID)
		assert.Equal(t, "basicos", prepared[0].Class.ClassCode)
	}
}

func TestPrepareACBatchAllowsRepeatedEntriesAndMarksExistingSolves(t *testing.T) {
	t.Parallel()
	for _, alreadyRecorded := range []bool{false, true} {
		t.Run(fmt.Sprint(alreadyRecorded), func(t *testing.T) {
			t.Parallel()
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			store.EXPECT().GetACValidationData([]data.ACKey{{UserID: "123", ProblemID: 42}, {UserID: "123", ProblemID: 42}}).Return(map[data.ACKey]data.ACValidationData{
				{UserID: "123", ProblemID: 42}: {Problem: &data.Problem{ID: 42, ClassID: 10}, Class: &data.Class{ID: 10, ClassCode: "basicos"}, AccessRoleIDs: []string{"student"}, AlreadyRecorded: alreadyRecorded},
			}, nil)
			prepared, issues, err := NewServices(store).Batch.PrepareACBatch([]service_model.CreateACParams{{UserID: "00123", ProblemID: 42}, {UserID: "123", ProblemID: 42}}, map[string]service_model.Principal{"123": {UserID: "123", Roles: []string{"student"}}})
			require.NoError(t, err)
			require.Empty(t, issues)
			require.Len(t, prepared, 2)
			for index, entry := range prepared {
				assert.Equal(t, index+1, entry.Entry)
				assert.Equal(t, "123", entry.User.UserID)
				assert.Equal(t, int64(42), entry.Problem.ID)
				assert.Equal(t, "basicos", entry.Class.ClassCode)
				assert.Equal(t, alreadyRecorded, entry.AlreadyRecorded)
			}
		})
	}
}

func TestPrepareACBatchDoesNotIgnoreOtherErrors(t *testing.T) {
	t.Parallel()
	for _, failure := range []error{errors.New("database unavailable"), serviceerr.NewConflictError("another conflict", nil)} {
		t.Run(failure.Error(), func(t *testing.T) {
			t.Parallel()
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			store.EXPECT().GetACValidationData([]data.ACKey{{UserID: "123", ProblemID: 42}}).Return(nil, failure)
			prepared, issues, err := NewServices(store).Batch.PrepareACBatch([]service_model.CreateACParams{{UserID: "123", ProblemID: 42}}, nil)
			assert.Nil(t, prepared)
			assert.Empty(t, issues, "technical failures are not entry validation issues")
			assert.IsType(t, &serviceerr.InternalError{}, err)
			assert.ErrorIs(t, err, failure)
		})
	}
}

func TestPrepareACBatchRejectsInvalidValuesWithoutLookups(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name  string
		input service_model.CreateACParams
	}{
		{name: "missing or null required values"},
		{name: "mention", input: service_model.CreateACParams{UserID: "<@123>", ProblemID: 42}},
		{name: "name", input: service_model.CreateACParams{UserID: "Ana", ProblemID: 42}},
		{name: "zero user ID", input: service_model.CreateACParams{UserID: "0", ProblemID: 42}},
		{name: "overflowing user ID", input: service_model.CreateACParams{UserID: "18446744073709551616", ProblemID: 42}},
		{name: "signed user ID", input: service_model.CreateACParams{UserID: "+123", ProblemID: 42}},
		{name: "negative problem", input: service_model.CreateACParams{UserID: "123", ProblemID: -1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			prepared, issues, err := NewServices(store).Batch.PrepareACBatch([]service_model.CreateACParams{test.input}, nil)
			require.NoError(t, err)
			assert.Nil(t, prepared)
			if assert.Len(t, issues, 1) {
				assert.Equal(t, 1, issues[0].Entry)
				assert.IsType(t, &serviceerr.ValidationError{}, issues[0].Err)
			}
		})
	}
}

func TestPrepareBatchRejectsInvalidSizeBeforeLookups(t *testing.T) {
	t.Parallel()
	for _, size := range []int{0, service_model.MaxBatchOperations + 1} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			t.Parallel()
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			service := NewServices(store).Batch
			problems, problemIssues := service.PrepareProblemBatch(make([]service_model.CreateProblemParams, size))
			acs, acIssues, err := service.PrepareACBatch(make([]service_model.CreateACParams, size), nil)
			require.NoError(t, err)
			assert.Nil(t, problems)
			assert.Nil(t, acs)
			for _, issues := range [][]service_model.BatchEntryError{problemIssues, acIssues} {
				if assert.Len(t, issues, 1) {
					assert.Equal(t, 0, issues[0].Entry)
					assert.IsType(t, &serviceerr.ValidationError{}, issues[0].Err)
				}
			}
		})
	}
}

func TestPrepareBatchDoesNotReturnPartialSuccess(t *testing.T) {
	t.Parallel()
	for _, problems := range []bool{true, false} {
		t.Run(fmt.Sprintf("problems=%t", problems), func(t *testing.T) {
			t.Parallel()
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			service := NewServices(store).Batch
			if problems {
				store.EXPECT().GetClassesByClassCodes([]string{"basicos"}).Return([]data.Class{{ID: 10, ClassCode: "basicos"}}, nil)
				prepared, issues := service.PrepareProblemBatch([]service_model.CreateProblemParams{
					{ClassCode: "basicos", Name: "Valid", URL: "https://example.com/problem"}, {},
				})
				assert.Nil(t, prepared)
				if assert.Len(t, issues, 1) {
					assert.Equal(t, 2, issues[0].Entry)
				}
			} else {
				store.EXPECT().GetACValidationData([]data.ACKey{{UserID: "123", ProblemID: 42}}).Return(map[data.ACKey]data.ACValidationData{{UserID: "123", ProblemID: 42}: {Problem: &data.Problem{ID: 42, ClassID: 10}, Class: &data.Class{ID: 10, ClassCode: "basicos"}, AccessRoleIDs: []string{"student"}, AlreadyRecorded: true}}, nil)

				prepared, issues, err := service.PrepareACBatch([]service_model.CreateACParams{{UserID: "123", ProblemID: 42}, {}}, map[string]service_model.Principal{"123": {UserID: "123", Roles: []string{"student"}}})
				require.NoError(t, err)
				assert.Nil(t, prepared)
				if assert.Len(t, issues, 1) {
					assert.Equal(t, 2, issues[0].Entry)
				}
			}
		})
	}
}

func TestPrepareACBatchFailsClosedOnLookupAndAccessErrors(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		facts   data.ACValidationData
		student service_model.Principal
		wantErr error
	}{
		{name: "missing problem", wantErr: &serviceerr.ProblemNotFoundError{}},
		{name: "missing class", facts: data.ACValidationData{Problem: &data.Problem{ID: 42, ClassID: 10}}, wantErr: &serviceerr.ClassCodeNotFoundError{}},
		{name: "no matching role", facts: data.ACValidationData{Problem: &data.Problem{ID: 42}, Class: &data.Class{ID: 10, ClassCode: "basicos"}, AccessRoleIDs: []string{"student"}}, student: service_model.Principal{UserID: "123", Roles: []string{"other"}}, wantErr: &serviceerr.UserNotEnrolledError{}},
		{name: "class has no roles", facts: data.ACValidationData{Problem: &data.Problem{ID: 42}, Class: &data.Class{ID: 10, ClassCode: "basicos"}}, student: service_model.Principal{UserID: "123", Roles: []string{"student"}}, wantErr: &serviceerr.UserNotEnrolledError{}},
		{name: "missing student", facts: data.ACValidationData{Problem: &data.Problem{ID: 42}, Class: &data.Class{ID: 10}}, wantErr: &serviceerr.InternalError{}},
		{name: "wrong student", facts: data.ACValidationData{Problem: &data.Problem{ID: 42}, Class: &data.Class{ID: 10}}, student: service_model.Principal{UserID: "456"}, wantErr: &serviceerr.InternalError{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			store.EXPECT().GetACValidationData([]data.ACKey{{UserID: "123", ProblemID: 42}}).Return(map[data.ACKey]data.ACValidationData{{UserID: "123", ProblemID: 42}: test.facts}, nil)
			prepared, issues, err := NewServices(store).Batch.PrepareACBatch([]service_model.CreateACParams{{UserID: "123", ProblemID: 42}}, map[string]service_model.Principal{"123": test.student})
			require.NoError(t, err)
			assert.Nil(t, prepared)
			require.Len(t, issues, 1)
			assert.Equal(t, 1, issues[0].Entry)
			assert.IsType(t, test.wantErr, issues[0].Err)
		})
	}
}

func TestPrepareProblemUpdateBatchValidatesTargetsWithoutWrites(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	store.EXPECT().GetProblemsByIDs([]int64{42}).Return([]data.Problem{{ID: 42}}, nil)
	store.EXPECT().GetClassesByClassCodes([]string{"basicos"}).Return([]data.Class{{ID: 10, ClassCode: "basicos"}}, nil)
	prepared, issues := NewServices(store).Batch.PrepareProblemUpdateBatch([]service_model.UpdateProblemParams{{ProblemID: 42, ClassCode: " basicos ", ProblemFields: service_model.ProblemFields{Name: " Updated ", URL: " https://example.com/new "}}})
	assert.Empty(t, issues)
	if assert.Len(t, prepared, 1) {
		assert.Equal(t, 1, prepared[0].Entry)
		assert.EqualValues(t, 42, prepared[0].Params.ProblemID)
		assert.Equal(t, "basicos", prepared[0].Params.ClassCode)
		assert.Equal(t, "Updated", prepared[0].Params.Name)
		assert.Equal(t, "https://example.com/new", prepared[0].Params.URL)
		assert.EqualValues(t, 10, prepared[0].Class.ID)
	}
}

func TestPrepareProblemUpdateBatchReportsIssuesWithoutPartialResults(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	store.EXPECT().GetProblemsByIDs([]int64{42, 99}).Return([]data.Problem{{ID: 42}}, nil)
	store.EXPECT().GetClassesByClassCodes([]string{"basicos"}).Return([]data.Class{{ID: 10, ClassCode: "basicos"}}, nil)
	prepared, issues := NewServices(store).Batch.PrepareProblemUpdateBatch([]service_model.UpdateProblemParams{
		{ProblemID: 42, ClassCode: "basicos", ProblemFields: service_model.ProblemFields{Name: "Valid", URL: "https://example.com/new"}},
		{},
		{ProblemID: 99, ClassCode: "basicos", ProblemFields: service_model.ProblemFields{Name: "Missing", URL: "https://example.com/new"}},
	})
	assert.Nil(t, prepared)
	if assert.Len(t, issues, 2) {
		assert.Equal(t, 2, issues[0].Entry)
		assert.Equal(t, 3, issues[1].Entry)
		assert.IsType(t, &serviceerr.ProblemNotFoundError{}, issues[1].Err)
	}
}

func TestProblemBatchUsesBulkLookupsForPreparationAndConfirmation(t *testing.T) {
	for _, size := range []int{1, service_model.MaxBatchOperations} {
		t.Run(fmt.Sprintf("%d entries", size), func(t *testing.T) {
			t.Parallel()

			t.Run("create", func(t *testing.T) {
				t.Parallel()
				store := mock_data.NewMockDataStore(gomock.NewController(t))
				classes := make([]data.Class, size)
				inputs := make([]service_model.CreateProblemParams, size)
				for index := range size {
					classCode := fmt.Sprintf("class-%03d", index)
					classes[index] = data.Class{ID: int64(index + 1), ClassCode: classCode}
					inputs[index] = service_model.CreateProblemParams{
						ClassCode: classCode,
						ProblemFields: service_model.ProblemFields{
							Name: fmt.Sprintf("Problem %d", index),
							URL:  "https://example.com/problem",
						},
					}
				}

				store.EXPECT().GetClassesByClassCodes(gomock.Any()).DoAndReturn(
					func(classCodes []string) ([]data.Class, error) {
						assert.Len(t, classCodes, size)
						return classes, nil
					},
				).Times(2)
				created := 0
				store.EXPECT().AddProblem(gomock.Any()).DoAndReturn(func(data.Problem) (int64, error) {
					created++
					return int64(created), nil
				}).Times(size)

				service := NewServices(store).Batch
				prepared, issues := service.PrepareProblemBatch(inputs)
				require.Empty(t, issues)
				require.Len(t, prepared, size)
				result, issues, err := service.ExecuteProblemBatch(prepared)
				require.NoError(t, err)
				require.Empty(t, issues)
				assert.Len(t, result.CreatedProblemIDs, size)
			})

			t.Run("update", func(t *testing.T) {
				t.Parallel()
				store := mock_data.NewMockDataStore(gomock.NewController(t))
				problems := make([]data.Problem, size)
				classes := make([]data.Class, size)
				inputs := make([]service_model.UpdateProblemParams, size)
				for index := range size {
					id := int64(index + 1)
					classCode := fmt.Sprintf("class-%03d", index)
					problems[index] = data.Problem{ID: id}
					classes[index] = data.Class{ID: id, ClassCode: classCode}
					inputs[index] = service_model.UpdateProblemParams{
						ProblemID: id,
						ClassCode: classCode,
						ProblemFields: service_model.ProblemFields{
							Name: fmt.Sprintf("Updated problem %d", index),
							URL:  "https://example.com/problem",
						},
					}
				}

				store.EXPECT().GetProblemsByIDs(gomock.Any()).DoAndReturn(
					func(problemIDs []int64) ([]data.Problem, error) {
						assert.Len(t, problemIDs, size)
						return problems, nil
					},
				).Times(2)
				store.EXPECT().GetClassesByClassCodes(gomock.Any()).DoAndReturn(
					func(classCodes []string) ([]data.Class, error) {
						assert.Len(t, classCodes, size)
						return classes, nil
					},
				).Times(2)
				store.EXPECT().UpdateProblemByProblemID(gomock.Any(), gomock.Any()).Return(nil).Times(size)

				service := NewServices(store).Batch
				prepared, issues := service.PrepareProblemUpdateBatch(inputs)
				require.Empty(t, issues)
				require.Len(t, prepared, size)
				result, issues, err := service.ExecuteProblemUpdateBatch(prepared)
				require.NoError(t, err)
				require.Empty(t, issues)
				assert.Len(t, result.UpdatedProblemIDs, size)
			})
		})
	}
}

func TestPrepareProblemUpdateBatchPreservesMissingTargetsWhenClassLookupFails(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	store.EXPECT().GetProblemsByIDs([]int64{42, 99}).Return([]data.Problem{{ID: 42}}, nil)
	store.EXPECT().GetClassesByClassCodes([]string{"basicos"}).Return(nil, errors.New("database unavailable"))

	prepared, issues := NewServices(store).Batch.PrepareProblemUpdateBatch([]service_model.UpdateProblemParams{
		{ProblemID: 42, ClassCode: "basicos", ProblemFields: service_model.ProblemFields{Name: "Existing", URL: "https://example.com/problem"}},
		{ProblemID: 99, ClassCode: "basicos", ProblemFields: service_model.ProblemFields{Name: "Missing", URL: "https://example.com/problem"}},
	})

	assert.Nil(t, prepared)
	if assert.Len(t, issues, 2) {
		assert.Equal(t, 1, issues[0].Entry)
		assert.IsType(t, &serviceerr.InternalError{}, issues[0].Err)
		assert.Equal(t, 2, issues[1].Entry)
		assert.IsType(t, &serviceerr.ProblemNotFoundError{}, issues[1].Err)
	}
}

func TestProblemUpdateBatchRejectsInvalidSizeBeforeReadingOrWriting(t *testing.T) {
	t.Parallel()
	for _, size := range []int{0, service_model.MaxBatchOperations + 1} {
		store := mock_data.NewMockDataStore(gomock.NewController(t))
		svc := NewServices(store).Batch
		prepared, issues := svc.PrepareProblemUpdateBatch(make([]service_model.UpdateProblemParams, size))
		assert.Nil(t, prepared)
		if assert.Len(t, issues, 1) {
			assert.Zero(t, issues[0].Entry)
		}
		result, issues, err := svc.ExecuteProblemUpdateBatch(make([]service_model.PreparedProblemUpdate, size))
		assert.NoError(t, err)
		assert.Empty(t, result.UpdatedProblemIDs)
		assert.Len(t, issues, 1)
	}
}

func TestACBatchUsesOneFactsReadForPreparationAndConfirmation(t *testing.T) {
	for _, size := range []int{1, service_model.MaxBatchOperations} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			t.Parallel()
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			inputs := make([]service_model.CreateACParams, size)
			targets := make([]data.ACKey, size)
			facts := make(map[data.ACKey]data.ACValidationData)
			students := make(map[string]service_model.Principal)
			for index := range size {
				id := int64(index + 1)
				code := fmt.Sprintf("class-%d", index)
				userID := fmt.Sprint(index + 1)
				inputs[index] = service_model.CreateACParams{UserID: userID, ProblemID: id}
				targets[index] = data.ACKey{UserID: userID, ProblemID: id}
				facts[targets[index]] = data.ACValidationData{Problem: &data.Problem{ID: id, ClassID: id}, Class: &data.Class{ID: id, ClassCode: code}, AccessRoleIDs: []string{code}}
				students[userID] = service_model.Principal{UserID: userID, Roles: []string{code}}
			}
			store.EXPECT().GetACValidationData(targets).Return(facts, nil).Times(2)
			store.EXPECT().AddACBatch(targets).Return(size, nil)
			batch := NewServices(store).Batch
			prepared, issues, err := batch.PrepareACBatch(inputs, students)
			require.NoError(t, err)
			require.Empty(t, issues)
			require.Len(t, prepared, size)
			for index, entry := range prepared {
				assert.Equal(t, index+1, entry.Entry)
				assert.Equal(t, inputs[index].UserID, entry.User.UserID)
				assert.Equal(t, inputs[index].ProblemID, entry.Problem.ID)
			}
			result, issues, err := batch.ExecuteACBatch(prepared, students)
			require.NoError(t, err)
			require.Empty(t, issues)
			assert.Equal(t, size, result.Recorded)
			assert.Zero(t, result.AlreadyRecorded)
		})
	}
}

func TestPrepareACBatchReportsAllBusinessIssuesInInputOrder(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	store.EXPECT().GetACValidationData([]data.ACKey{{UserID: "123", ProblemID: 42}, {UserID: "123", ProblemID: 99}}).Return(map[data.ACKey]data.ACValidationData{
		{UserID: "123", ProblemID: 42}: {Problem: &data.Problem{ID: 42, ClassID: 10}},
	}, nil)
	prepared, issues, err := NewServices(store).Batch.PrepareACBatch([]service_model.CreateACParams{{UserID: "123", ProblemID: 42}, {UserID: "123", ProblemID: 99}, {}}, nil)
	require.NoError(t, err)
	assert.Nil(t, prepared)
	require.Len(t, issues, 3)
	assert.IsType(t, &serviceerr.ClassCodeNotFoundError{}, issues[0].Err)
	assert.IsType(t, &serviceerr.ProblemNotFoundError{}, issues[1].Err)
	assert.IsType(t, &serviceerr.ValidationError{}, issues[2].Err)
	for index, issue := range issues {
		assert.Equal(t, index+1, issue.Entry)
	}
}

func TestPrepareACBatchDoesNotShareAccessRolesBetweenClasses(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	store.EXPECT().GetACValidationData([]data.ACKey{{UserID: "123", ProblemID: 42}, {UserID: "123", ProblemID: 43}}).Return(map[data.ACKey]data.ACValidationData{
		{UserID: "123", ProblemID: 42}: {Problem: &data.Problem{ID: 42}, Class: &data.Class{ID: 10, ClassCode: "first"}, AccessRoleIDs: []string{"role-first"}},
		{UserID: "123", ProblemID: 43}: {Problem: &data.Problem{ID: 43}, Class: &data.Class{ID: 20, ClassCode: "second"}, AccessRoleIDs: []string{"role-second"}},
	}, nil)
	prepared, issues, err := NewServices(store).Batch.PrepareACBatch([]service_model.CreateACParams{{UserID: "123", ProblemID: 42}, {UserID: "123", ProblemID: 43}}, map[string]service_model.Principal{"123": {UserID: "123", Roles: []string{"role-first"}}})
	require.NoError(t, err)
	assert.Nil(t, prepared)
	require.Len(t, issues, 1)
	assert.Equal(t, 2, issues[0].Entry)
	assert.IsType(t, &serviceerr.UserNotEnrolledError{}, issues[0].Err)
}
