package adapters

import (
	"testing"

	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
)

func TestPriorityThroughPresentationAdapters(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		priority *string
		label    string
	}{
		{name: "assigned", priority: new("recommended"), label: "Recommended"},
		{name: "unassigned", label: "No priority assigned"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			groups := []service_model.ProblemsByTopic{{
				Topic:    "Arrays",
				Problems: []service_model.Problem{{ID: 7, Name: "Two Sum", Priority: tt.priority}},
			}}
			assert.Equal(t, tt.priority, TransformToPresentationProblemsByTopic(groups)[0].Problems[0].Priority)
			assert.Equal(t, tt.priority, TransformToExcelizerProblemsByTopic(groups)[0].Problems[0].Priority)

			table := ScoreToTableScore(service_model.Score{
				ProblemsByTopic: groups,
				UserScores: []service_model.PairUserProblemScore{{
					User:          service_model.User{UserID: "student", DisplayName: "Student"},
					CountProblems: 1,
					Problems:      map[int64]bool{7: true},
				}},
			})
			assert.Equal(t, []string{"(ID) Problem \\ User", "Student"}, table.Header)
			assert.Equal(t, []string{"Total", "1"}, table.Table[0])
			assert.Equal(t, []string{"(7) Two Sum [Priority: " + tt.label + "]", "AC"}, table.Table[2])
			assert.Equal(t, tt.priority, table.RowMetadata[2].Problem.Priority)
			assert.True(t, table.RowMetadata[2].CellMetadata[1].IsAC)
		})
	}
}
