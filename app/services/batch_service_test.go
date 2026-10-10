package services

import (
	"errors"
	"fmt"
	"testing"

	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
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
	store.EXPECT().GetProblemByID(int64(42)).Return(data.Problem{ID: 42, ClassID: 10, Name: "Two Sum"}, nil)
	store.EXPECT().GetClassByClassID(int64(10)).Return(data.Class{ID: 10, ClassCode: "basicos"}, nil)
	store.EXPECT().HasUserSolvedProblem("123", int64(42)).Return(false, nil)
	store.EXPECT().GetClassAccessRolesByClassCode("basicos").Return([]data.ClassAccessRole{{RoleID: "student"}}, nil)
	prepared, issues := NewServices(store).Batch.PrepareACBatch([]service_model.CreateACParams{{UserID: " 00123 ", ProblemID: 42}}, map[string]service_model.Principal{"123": {UserID: "123", DisplayName: "Ana", Roles: []string{"student"}}})
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
	for _, test := range []struct {
		name            string
		alreadyRecorded bool
	}{
		{name: "repeated new ACs remain valid"},
		{name: "existing ACs are marked as omitted", alreadyRecorded: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			store.EXPECT().GetProblemByID(int64(42)).Return(data.Problem{ID: 42, ClassID: 10}, nil).Times(2)
			store.EXPECT().GetClassByClassID(int64(10)).Return(data.Class{ID: 10, ClassCode: "basicos"}, nil).Times(2)
			store.EXPECT().HasUserSolvedProblem("123", int64(42)).Return(test.alreadyRecorded, nil).Times(2)
			store.EXPECT().GetClassAccessRolesByClassCode("basicos").Return([]data.ClassAccessRole{{RoleID: "student"}}, nil).Times(2)
			prepared, issues := NewServices(store).Batch.PrepareACBatch([]service_model.CreateACParams{{UserID: "00123", ProblemID: 42}, {UserID: "123", ProblemID: 42}}, map[string]service_model.Principal{"123": {UserID: "123", Roles: []string{"student"}}})
			assert.Empty(t, issues)
			if assert.Len(t, prepared, 2) {
				for index, entry := range prepared {
					assert.Equal(t, index+1, entry.Entry)
					assert.Equal(t, "123", entry.User.UserID)
					assert.Equal(t, int64(42), entry.Problem.ID)
					assert.Equal(t, "basicos", entry.Class.ClassCode)
					assert.Equal(t, test.alreadyRecorded, entry.AlreadyRecorded)
				}
			}
		})
	}
}

func TestPrepareACBatchDoesNotIgnoreOtherErrors(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		queryErr error
	}{
		{name: "database failure", queryErr: errors.New("database unavailable")},
		{name: "generic conflict", queryErr: serviceerr.NewConflictError("another conflict", nil)},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			store.EXPECT().GetProblemByID(int64(42)).Return(data.Problem{ID: 42, ClassID: 10}, nil)
			store.EXPECT().GetClassByClassID(int64(10)).Return(data.Class{ID: 10, ClassCode: "basicos"}, nil)
			store.EXPECT().GetClassAccessRolesByClassCode("basicos").Return([]data.ClassAccessRole{{RoleID: "student"}}, nil)
			store.EXPECT().HasUserSolvedProblem("123", int64(42)).Return(false, test.queryErr)
			prepared, issues := NewServices(store).Batch.PrepareACBatch([]service_model.CreateACParams{{UserID: "123", ProblemID: 42}}, map[string]service_model.Principal{"123": {UserID: "123", Roles: []string{"student"}}})
			assert.Nil(t, prepared)
			if assert.Len(t, issues, 1) {
				assert.Equal(t, 1, issues[0].Entry)
				assert.IsType(t, &serviceerr.InternalError{}, issues[0].Err)
				assert.ErrorIs(t, issues[0].Err, test.queryErr)
			}
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
			prepared, issues := NewServices(store).Batch.PrepareACBatch([]service_model.CreateACParams{test.input}, nil)
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
			acs, acIssues := service.PrepareACBatch(make([]service_model.CreateACParams, size), nil)
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
				store.EXPECT().GetProblemByID(int64(42)).Return(data.Problem{ID: 42, ClassID: 10}, nil)
				store.EXPECT().GetClassByClassID(int64(10)).Return(data.Class{ID: 10, ClassCode: "basicos"}, nil)
				store.EXPECT().HasUserSolvedProblem("123", int64(42)).Return(true, nil)
				store.EXPECT().GetClassAccessRolesByClassCode("basicos").Return([]data.ClassAccessRole{{RoleID: "student"}}, nil)
				prepared, issues := service.PrepareACBatch([]service_model.CreateACParams{{UserID: "123", ProblemID: 42}, {}}, map[string]service_model.Principal{"123": {UserID: "123", Roles: []string{"student"}}})
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
		name           string
		setup          func(*mock_data.MockDataStore)
		principal      service_model.Principal
		accessRoles    []data.ClassAccessRole
		studentMissing bool
		wantType       error
	}{
		{name: "problem missing", setup: func(m *mock_data.MockDataStore) {
			m.EXPECT().GetProblemByID(int64(42)).Return(data.Problem{}, dataerr.NewNotFoundError("gone", nil))
		}, wantType: &serviceerr.ProblemNotFoundError{}},
		{name: "problem query failed", setup: func(m *mock_data.MockDataStore) {
			m.EXPECT().GetProblemByID(int64(42)).Return(data.Problem{}, errors.New("database unavailable"))
		}, wantType: &serviceerr.InternalError{}},
		{name: "student has no matching role", principal: service_model.Principal{UserID: "123", Roles: []string{"other"}}, accessRoles: []data.ClassAccessRole{{RoleID: "student"}}, wantType: &serviceerr.UserNotEnrolledError{}},
		{name: "class has no access roles", principal: service_model.Principal{UserID: "123", Roles: []string{"student"}}, wantType: &serviceerr.UserNotEnrolledError{}},
		{name: "student data missing", studentMissing: true, wantType: &serviceerr.InternalError{}},
		{name: "wrong student identity", principal: service_model.Principal{UserID: "456", Roles: []string{"student"}}, wantType: &serviceerr.InternalError{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			if test.setup != nil {
				test.setup(store)
			} else {
				store.EXPECT().GetProblemByID(int64(42)).Return(data.Problem{ID: 42, ClassID: 10}, nil)
				store.EXPECT().GetClassByClassID(int64(10)).Return(data.Class{ID: 10, ClassCode: "basicos"}, nil)
				if !test.studentMissing && test.principal.UserID == "123" {
					store.EXPECT().GetClassAccessRolesByClassCode("basicos").Return(test.accessRoles, nil)
				}
			}
			students := map[string]service_model.Principal{"123": test.principal}
			if test.studentMissing {
				students = nil
			}
			prepared, issues := NewServices(store).Batch.PrepareACBatch([]service_model.CreateACParams{{UserID: "123", ProblemID: 42}}, students)
			assert.Nil(t, prepared)
			if assert.Len(t, issues, 1) {
				assert.IsType(t, test.wantType, issues[0].Err)
			}
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
