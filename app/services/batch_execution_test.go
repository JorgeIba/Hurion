package services

import (
	"fmt"
	"testing"

	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
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
	problem.EXPECT().PrepareCreation(params).Return(service_model.PreparedProblem{
		Params: params, Class: service_model.Class{ID: 20, ClassCode: "basicos"},
	}, nil)
	result, issues, err := (&BatchService{Problem: problem}).ExecuteProblemBatch([]service_model.PreparedProblem{
		{Entry: 1, Params: params, Class: service_model.Class{ID: 10, ClassCode: "basicos"}},
	})
	require.NoError(t, err)
	assert.Empty(t, result.CreatedProblemIDs)
	require.Len(t, issues, 1)
	assert.Equal(t, 1, issues[0].Entry)
	assert.IsType(t, &serviceerr.ConflictError{}, issues[0].Err)
}
