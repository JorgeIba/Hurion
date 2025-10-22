package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProblem_GetDifficulty_WithValue(t *testing.T) {
	// Precondition
	difficulty := string(DifficultyHard)
	problem := Problem{
		Name:       "Test Problem",
		Difficulty: &difficulty,
	}

	// Act
	result := problem.GetDifficulty()

	// Verify
	assert.Equal(t, DifficultyHard, result)
}

func TestProblem_GetDifficulty_WithoutValue(t *testing.T) {
	// Precondition
	problem := Problem{
		Name:       "Test Problem",
		Difficulty: nil,
	}

	// Act
	result := problem.GetDifficulty()

	// Verify
	assert.Equal(t, DifficultyDefault, result)
}

func TestProblem_GetTopic_WithValue(t *testing.T) {
	// Precondition
	topic := "Dynamic Programming"
	problem := Problem{
		Name:  "Test Problem",
		Topic: &topic,
	}

	// Act
	result := problem.GetTopic()

	// Verify
	assert.Equal(t, "Dynamic Programming", result)
}

func TestProblem_GetTopic_WithoutValue(t *testing.T) {
	// Precondition
	problem := Problem{
		Name:  "Test Problem",
		Topic: nil,
	}

	// Act
	result := problem.GetTopic()

	// Verify
	assert.Equal(t, DefaultTopic, result)
}

func TestIsValidDifficulty(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Valid - Default", "Not specified", true},
		{"Valid - Easy", "Easy", true},
		{"Valid - Medium", "Medium", true},
		{"Valid - Hard", "Hard", true},
		{"Invalid - lowercase", "easy", false},
		{"Invalid - random", "Super Hard", false},
		{"Invalid - empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidDifficulty(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseDifficulty(t *testing.T) {
	// Test valid difficulties
	validTests := []struct {
		name     string
		input    string
		expected Difficulty
	}{
		{"Parse Default", "Not specified", DifficultyDefault},
		{"Parse Easy", "Easy", DifficultyEasy},
		{"Parse Medium", "Medium", DifficultyMedium},
		{"Parse Hard", "Hard", DifficultyHard},
	}

	for _, tt := range validTests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDifficulty(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}

	// Test invalid difficulty
	t.Run("Parse Invalid", func(t *testing.T) {
		_, err := ParseDifficulty("Invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid difficulty")
	})
}

func TestGetValidDifficulties(t *testing.T) {
	// Act
	difficulties := GetValidDifficulties()

	// Verify
	expected := []Difficulty{DifficultyDefault, DifficultyEasy, DifficultyMedium, DifficultyHard}
	assert.Len(t, difficulties, 4)
	assert.Equal(t, expected, difficulties)
}

func TestDifficulty_String(t *testing.T) {
	tests := []struct {
		name       string
		difficulty Difficulty
		expected   string
	}{
		{"Default to string", DifficultyDefault, "Not specified"},
		{"Easy to string", DifficultyEasy, "Easy"},
		{"Medium to string", DifficultyMedium, "Medium"},
		{"Hard to string", DifficultyHard, "Hard"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.difficulty.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}
