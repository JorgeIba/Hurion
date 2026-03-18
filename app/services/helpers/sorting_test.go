package helpers

import (
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/stretchr/testify/assert"
)

func TestGetDifficultyOrder(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 1, GetDifficultyOrder(data.DifficultyEasy))
	assert.Equal(t, 2, GetDifficultyOrder(data.DifficultyMedium))
	assert.Equal(t, 3, GetDifficultyOrder(data.DifficultyHard))
	assert.Equal(t, 4, GetDifficultyOrder(data.DifficultyDefault))
	assert.Equal(t, 5, GetDifficultyOrder("Unknown"))
}

func TestSortProblemsByTopicAndDifficulty(t *testing.T) {
	t.Parallel()
	problems := []data.Problem{
		{ID: 1, Topic: PtrStr("B"), Difficulty: PtrStr("Hard")},
		{ID: 2, Topic: PtrStr("A"), Difficulty: PtrStr("Medium")},
		{ID: 3, Topic: PtrStr("A"), Difficulty: PtrStr("Easy")},
	}

	SortProblemsByTopicAndDifficulty(problems)

	// Topic A comes before B.
	// Within A, Easy comes before Medium.
	assert.Equal(t, int64(3), problems[0].ID) // A - Easy
	assert.Equal(t, int64(2), problems[1].ID) // A - Medium
	assert.Equal(t, int64(1), problems[2].ID) // B - Hard
}

func TestSortScore(t *testing.T) {
	t.Parallel()
	u1 := data.User{UserID: "u1"}
	u2 := data.User{UserID: "u2"}
	
	scores := map[data.User][]data.SolvedProblem{
		u1: {{ProblemID: 1}},
		u2: {{ProblemID: 1}, {ProblemID: 2}},
	}

	sorted := SortScore(scores)

	assert.Equal(t, "u2", sorted[0].User.UserID)
	assert.Equal(t, "u1", sorted[1].User.UserID)
}
