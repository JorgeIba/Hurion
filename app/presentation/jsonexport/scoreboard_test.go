package jsonexport

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatScoreboard(t *testing.T) {
	diff := "easy"
	topic := "Arrays"
	createdAt := time.Date(2026, 10, 4, 12, 30, 0, 0, time.FixedZone("offset", -6*60*60))
	score := service_model.Score{
		ProblemsByTopic: []service_model.ProblemsByTopic{
			{
				Topic: topic,
				Problems: []service_model.Problem{
					{
						ID:         101,
						Name:       "Two Sum",
						URL:        "http://example.com/101",
						Difficulty: &diff,
						Topic:      &topic,
						CreatedAt:  createdAt,
					},
				},
			},
		},
		UserScores: []service_model.PairUserProblemScore{
			{
				User: service_model.User{
					UserID:      "u123",
					DisplayName: "Alice",
				},
				CountProblems: 1,
				Problems: map[int64]bool{
					101: true,
				},
			},
		},
	}

	data, err := FormatScoreboard("cs101", "Computer Science 101", score)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	var result ScoreboardJSON
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, "cs101", result.ClassCode)
	assert.Equal(t, "Computer Science 101", result.ClassName)
	require.Len(t, result.Problems, 1)
	assert.Equal(t, int64(101), result.Problems[0].ID)
	assert.Equal(t, "Two Sum", result.Problems[0].Name)
	assert.Equal(t, &diff, result.Problems[0].Difficulty)
	assert.Equal(t, createdAt.UTC(), result.Problems[0].CreatedAt)
	assert.Contains(t, string(data), `"createdAt": "2026-10-04T18:30:00Z"`)

	require.Len(t, result.Users, 1)
	assert.Equal(t, "u123", result.Users[0].UserID)
	assert.Equal(t, []int64{101}, result.Users[0].Solved)
}

func TestFormatScoreboard_EmptyClass_ReturnsEmptyJSONArrays(t *testing.T) {
	// WHY THIS TEST MATTERS: The web frontend expects JSON arrays (`[]`), not `null`.
	// If uninitialized slices format as `null`, frontend `problems.map(...)` throws a TypeError.
	score := service_model.Score{
		ProblemsByTopic: nil,
		UserScores:      nil,
	}

	data, err := FormatScoreboard("empty_101", "Empty Class", score)
	require.NoError(t, err)

	jsonStr := string(data)
	assert.True(t, strings.Contains(jsonStr, `"problems": []`), "problems should format as [] instead of null")
	assert.True(t, strings.Contains(jsonStr, `"users": []`), "users should format as [] instead of null")
}

func TestFormatScoreboard_MultipleSolvedProblems_AggregatesSolvedIDs(t *testing.T) {
	// WHY THIS TEST MATTERS: Verifies that unsolved problems (false) are excluded,
	// while solved problems (true) are collected into the user's solved ID slice.
	score := service_model.Score{
		UserScores: []service_model.PairUserProblemScore{
			{
				User: service_model.User{
					UserID:      "u456",
					DisplayName: "Bob",
				},
				CountProblems: 2,
				Problems: map[int64]bool{
					101: true,
					102: false,
					103: true,
				},
			},
		},
	}

	data, err := FormatScoreboard("cs102", "CS 102", score)
	require.NoError(t, err)

	var result ScoreboardJSON
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	require.Len(t, result.Users, 1)
	assert.Equal(t, 2, result.Users[0].SolvedCount)
	assert.Equal(t, []int64{101, 103}, result.Users[0].Solved)
}

func TestFormatScoreboard_SolvedIDsAreDeterministic(t *testing.T) {
	newScore := func(problems map[int64]bool) service_model.Score {
		return service_model.Score{
			UserScores: []service_model.PairUserProblemScore{{
				User:          service_model.User{UserID: "u456", DisplayName: "Bob"},
				CountProblems: 3,
				Problems:      problems,
			}},
		}
	}

	first, err := FormatScoreboard("cs102", "CS 102", newScore(map[int64]bool{
		103: true,
		101: true,
		102: true,
	}))
	require.NoError(t, err)

	second, err := FormatScoreboard("cs102", "CS 102", newScore(map[int64]bool{
		102: true,
		103: true,
		101: true,
	}))
	require.NoError(t, err)

	assert.Equal(t, first, second)
}
