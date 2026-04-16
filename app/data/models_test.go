package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProblem_GetDifficulty(t *testing.T) {
	t.Parallel()

	easy := "Easy"
	tests := []struct {
		name       string
		problem    Problem
		assertFunc func(*testing.T, Difficulty)
	}{
		{
			name: "with value",
			problem: Problem{
				Name:       "Test Problem",
				Difficulty: &easy,
			},
			assertFunc: func(t *testing.T, result Difficulty) {
				assert.Equal(t, Difficulty("Easy"), result)
			},
		},
		{
			name: "without value",
			problem: Problem{
				Name:       "Test Problem",
				Difficulty: nil,
			},
			assertFunc: func(t *testing.T, result Difficulty) {
				assert.Equal(t, DifficultyDefault, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.problem.GetDifficulty()
			tt.assertFunc(t, result)
		})
	}
}

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

func TestIsValidDifficulty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		assertFunc func(*testing.T, bool)
	}{
		{"Valid - Default", "Not specified", func(t *testing.T, b bool) { assert.True(t, b) }},
		{"Valid - Easy", "Easy", func(t *testing.T, b bool) { assert.True(t, b) }},
		{"Valid - Medium", "Medium", func(t *testing.T, b bool) { assert.True(t, b) }},
		{"Valid - Hard", "Hard", func(t *testing.T, b bool) { assert.True(t, b) }},
		{"Invalid - lowercase", "easy", func(t *testing.T, b bool) { assert.False(t, b) }},
		{"Invalid - random", "Super Hard", func(t *testing.T, b bool) { assert.False(t, b) }},
		{"Invalid - empty", "", func(t *testing.T, b bool) { assert.False(t, b) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := IsValidDifficulty(tt.input)
			tt.assertFunc(t, result)
		})
	}
}

func TestParseDifficulty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		assertFunc func(*testing.T, Difficulty, error)
	}{
		{
			name:  "Parse Default",
			input: "Not specified",
			assertFunc: func(t *testing.T, d Difficulty, err error) {
				assert.NoError(t, err)
				assert.Equal(t, DifficultyDefault, d)
			},
		},
		{
			name:  "Parse Easy",
			input: "Easy",
			assertFunc: func(t *testing.T, d Difficulty, err error) {
				assert.NoError(t, err)
				assert.Equal(t, DifficultyEasy, d)
			},
		},
		{
			name:  "Parse Medium",
			input: "Medium",
			assertFunc: func(t *testing.T, d Difficulty, err error) {
				assert.NoError(t, err)
				assert.Equal(t, DifficultyMedium, d)
			},
		},
		{
			name:  "Parse Hard",
			input: "Hard",
			assertFunc: func(t *testing.T, d Difficulty, err error) {
				assert.NoError(t, err)
				assert.Equal(t, DifficultyHard, d)
			},
		},
		{
			name:  "Parse Invalid",
			input: "Invalid",
			assertFunc: func(t *testing.T, d Difficulty, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid difficulty")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ParseDifficulty(tt.input)
			tt.assertFunc(t, result, err)
		})
	}
}

func TestGetValidDifficulties(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		assertFunc func(*testing.T, []Difficulty)
	}{
		{
			name: "get all",
			assertFunc: func(t *testing.T, difficulties []Difficulty) {
				expected := []Difficulty{DifficultyDefault, DifficultyEasy, DifficultyMedium, DifficultyHard}
				assert.Len(t, difficulties, 4)
				assert.Equal(t, expected, difficulties)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := GetValidDifficulties()
			tt.assertFunc(t, result)
		})
	}
}

func TestDifficulty_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		difficulty Difficulty
		assertFunc func(*testing.T, string)
	}{
		{"Default to string", DifficultyDefault, func(t *testing.T, s string) { assert.Equal(t, "Not specified", s) }},
		{"Easy to string", DifficultyEasy, func(t *testing.T, s string) { assert.Equal(t, "Easy", s) }},
		{"Medium to string", DifficultyMedium, func(t *testing.T, s string) { assert.Equal(t, "Medium", s) }},
		{"Hard to string", DifficultyHard, func(t *testing.T, s string) { assert.Equal(t, "Hard", s) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.difficulty.String()
			tt.assertFunc(t, result)
		})
	}
}
