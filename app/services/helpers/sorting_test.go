package helpers

import (
	"testing"

	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
)

func TestSortProblemsByID(t *testing.T) {
	t.Parallel()

	problems := []service_model.Problem{
		{ID: 4, Topic: new("Strings"), Difficulty: new("easy")},
		{ID: 2, Topic: new("Arrays"), Difficulty: new("hard")},
		{ID: 3, Topic: new("Arrays")},
		{ID: 1, Topic: new("Strings"), Difficulty: new("medium")},
	}

	SortProblemsByID(problems)

	assert.Equal(t, int64(1), problems[0].ID)
	assert.Equal(t, int64(2), problems[1].ID)
	assert.Equal(t, int64(3), problems[2].ID)
	assert.Equal(t, int64(4), problems[3].ID)
}

func TestSortScore(t *testing.T) {
	t.Parallel()

	userScores := []service_model.PairUserProblemScore{
		{User: service_model.User{UserID: "u1"}, CountProblems: 2},
		{User: service_model.User{UserID: "u2"}, CountProblems: 5},
		{User: service_model.User{UserID: "u3"}, CountProblems: 1},
	}

	SortScore(userScores)

	assert.Equal(t, "u2", userScores[0].User.UserID)
	assert.Equal(t, "u1", userScores[1].User.UserID)
	assert.Equal(t, "u3", userScores[2].User.UserID)
}
