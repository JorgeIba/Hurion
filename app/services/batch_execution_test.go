package services

import (
	"errors"
	"fmt"
	"testing"

	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/services/mock_services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestExecuteBatchRejectsInvalidSizeBeforeLookups(t *testing.T) {
	t.Parallel()
	for _, size := range []int{0, service_model.MaxBatchOperations + 1} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			t.Parallel()
			service := &BatchService{}
			problems, problemIssues, err := service.ExecuteProblemBatch(make([]service_model.PreparedProblem, size))
			require.NoError(t, err)
			acs, acIssues, err := service.ExecuteACBatch(make([]service_model.PreparedAC, size), nil)
			require.NoError(t, err)
			assert.Empty(t, problems.CreatedProblemIDs)
			assert.Zero(t, acs.Recorded)
			for _, issues := range [][]service_model.BatchEntryError{problemIssues, acIssues} {
				require.Len(t, issues, 1)
				assert.Zero(t, issues[0].Entry)
				assert.IsType(t, &serviceerr.ValidationError{}, issues[0].Err)
			}
		})
	}
}

func TestExecuteProblemBatchRejectsReplacedClassBeforeWrites(t *testing.T) {
	t.Parallel()
	problem := mock_services.NewMockProblemService(gomock.NewController(t))
	params := service_model.CreateProblemParams{ClassCode: "basicos", Name: "New", URL: "https://example.com"}
	problem.EXPECT().ExecuteCreationBatch(gomock.Any()).Return(
		service_model.ProblemCreateBatchResult{},
		[]service_model.BatchEntryError{{Entry: 1, Err: serviceerr.NewConflictError("destination class changed", nil)}},
		nil,
	)
	result, issues, err := (&BatchService{Problem: problem}).ExecuteProblemBatch([]service_model.PreparedProblem{
		{Entry: 1, Params: params, Class: service_model.Class{ID: 10, ClassCode: "basicos"}},
	})
	require.NoError(t, err)
	assert.Empty(t, result.CreatedProblemIDs)
	require.Len(t, issues, 1)
	assert.Equal(t, 1, issues[0].Entry)
	assert.IsType(t, &serviceerr.ConflictError{}, issues[0].Err)
}

func TestExecuteACBatchCountsPairsAndEnsuresEachStudentOnce(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	store.EXPECT().GetACValidationData([]data.ACKey{{UserID: "123", ProblemID: 42}, {UserID: "123", ProblemID: 42}, {UserID: "123", ProblemID: 43}, {UserID: "456", ProblemID: 42}}).Return(map[data.ACKey]data.ACValidationData{
		{UserID: "123", ProblemID: 42}: {Problem: &data.Problem{ID: 42}, Class: &data.Class{ID: 10}, AccessRoleIDs: []string{"student"}},
		{UserID: "123", ProblemID: 43}: {Problem: &data.Problem{ID: 43}, Class: &data.Class{ID: 10}, AccessRoleIDs: []string{"student"}},
		{UserID: "456", ProblemID: 42}: {Problem: &data.Problem{ID: 42}, Class: &data.Class{ID: 10}, AccessRoleIDs: []string{"student"}, AlreadyRecorded: true},
	}, nil)

	store.EXPECT().AddACBatch([]data.ACKey{
		{UserID: "123", ProblemID: 42}, {UserID: "123", ProblemID: 42},
		{UserID: "123", ProblemID: 43}, {UserID: "456", ProblemID: 42},
	}).Return(2, nil)
	batch := []service_model.PreparedAC{
		{User: service_model.User{UserID: "123"}, Problem: service_model.Problem{ID: 42}, Class: service_model.Class{ID: 10}},
		{User: service_model.User{UserID: "123"}, Problem: service_model.Problem{ID: 42}, Class: service_model.Class{ID: 10}},
		{User: service_model.User{UserID: "123"}, Problem: service_model.Problem{ID: 43}, Class: service_model.Class{ID: 10}},
		{User: service_model.User{UserID: "456"}, Problem: service_model.Problem{ID: 42}, Class: service_model.Class{ID: 10}},
	}
	students := map[string]service_model.Principal{
		"123": {UserID: "123", Roles: []string{"student"}},
		"456": {UserID: "456", Roles: []string{"student"}},
	}
	result, issues, err := NewServices(store).Batch.ExecuteACBatch(batch, students)
	require.NoError(t, err)
	assert.Empty(t, issues)
	assert.Equal(t, 2, result.Recorded)
	assert.Equal(t, 2, result.AlreadyRecorded)
}

func TestExecuteACBatchReadFailureStopsBeforeWrites(t *testing.T) {
	t.Parallel()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	failure := errors.New("validation read unavailable")
	store.EXPECT().GetACValidationData([]data.ACKey{{UserID: "123", ProblemID: 42}}).Return(nil, failure)
	result, issues, err := NewServices(store).Batch.ExecuteACBatch([]service_model.PreparedAC{
		{User: service_model.User{UserID: "123"}, Problem: service_model.Problem{ID: 42}, Class: service_model.Class{ID: 10}},
	}, nil)
	assert.Zero(t, result)
	assert.Empty(t, issues)
	assert.ErrorIs(t, err, failure)
	_, preparationFailed := errors.AsType[*serviceerr.BatchPreparationError](err)
	assert.True(t, preparationFailed, "callers must distinguish a failed read from an uncertain write")
}
