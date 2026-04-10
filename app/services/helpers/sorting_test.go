package helpers

import (
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
	"github.com/stretchr/testify/assert"
)

func TestGetDifficultyOrder(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 1, GetDifficultyOrder("Easy"))
	assert.Equal(t, 2, GetDifficultyOrder("Medium"))
	assert.Equal(t, 3, GetDifficultyOrder("Hard"))
	assert.Equal(t, 4, GetDifficultyOrder(service_model.DifficultyDefault))
	assert.Equal(t, 5, GetDifficultyOrder("Unknown"))
}

func TestSortProblemsByTopicAndDifficulty(t *testing.T) {
	t.Parallel()

	problems := []service_model.Problem{
		{ID: 1, Topic: ptrStr("Strings"), Difficulty: ptrStr("Medium")},
		{ID: 2, Topic: ptrStr("Arrays"), Difficulty: ptrStr("Hard")},
		{ID: 3, Topic: ptrStr("Arrays"), Difficulty: ptrStr("Easy")},
		{ID: 4, Topic: ptrStr("Strings"), Difficulty: ptrStr("Easy")},
	}

	SortProblemsByTopicAndDifficulty(problems)

	// Order should be:
	// 1. Arrays, Easy (ID 3)
	// 2. Arrays, Hard (ID 2)
	// 3. Strings, Easy (ID 4)
	// 4. Strings, Medium (ID 1)
	assert.Equal(t, int64(3), problems[0].ID)
	assert.Equal(t, int64(2), problems[1].ID)
	assert.Equal(t, int64(4), problems[2].ID)
	assert.Equal(t, int64(1), problems[3].ID)
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

func ptrStr(s string) *string { return &s }
