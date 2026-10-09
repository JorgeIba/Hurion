package services

import (
	"testing"

	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewServices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{
			name: "success - all services wired correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_data.NewMockDataStore(ctrl)

			services := NewServices(mockStore)

			assert.NotNil(t, services)

			// Verify Datastore is correctly injected into all services
			assert.Equal(t, mockStore, services.User.Datastore)
			assert.Equal(t, mockStore, services.Enrollment.Datastore)
			assert.Equal(t, mockStore, services.Class.Datastore)
			assert.Equal(t, mockStore, services.Solution.Datastore)
			assert.Equal(t, mockStore, services.Score.Datastore)
			assert.Same(t, services.Problem, services.Batch.Problem)
			assert.Same(t, services.Solution, services.Batch.Solution)

			// Verify the problemService struct inside the interface has the correct Datastore
			concreteProblemSvc, ok := services.Problem.(*problemService)
			assert.True(t, ok, "ProblemService should be of type *problemService")
			if ok {
				assert.Equal(t, mockStore, concreteProblemSvc.Datastore)
			}

			// Verify ScoreService is wired with the exact same ProblemService instance
			assert.Equal(t, services.Problem, services.Score.ProblemService)
		})
	}
}
