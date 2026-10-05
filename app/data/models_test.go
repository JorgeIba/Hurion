package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProblem_GetTopic(t *testing.T) {
	t.Parallel()

	dp := "Dynamic Programming"
	tests := []struct {
		name       string
		problem    Problem
		assertFunc func(*testing.T, string)
	}{
		{
			name: "with value",
			problem: Problem{
				Name:  "Test Problem",
				Topic: &dp,
			},
			assertFunc: func(t *testing.T, result string) {
				assert.Equal(t, "Dynamic Programming", result)
			},
		},
		{
			name: "without value",
			problem: Problem{
				Name:  "Test Problem",
				Topic: nil,
			},
			assertFunc: func(t *testing.T, result string) {
				assert.Equal(t, DefaultTopic, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.problem.GetTopic()
			tt.assertFunc(t, result)
		})
	}
}
