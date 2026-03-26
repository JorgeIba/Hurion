package discord

import (
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

func TestFormatProblemsAsEmbeds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		classCode       string
		groupedProblems []services.ProblemsByTopic
		assertFunc      func(t *testing.T, res []*discordgo.MessageEmbed)
	}{
		{
			name:      "Empty problems",
			classCode: "CS101",
			groupedProblems: []services.ProblemsByTopic{},
			assertFunc: func(t *testing.T, res []*discordgo.MessageEmbed) {
				assert.Empty(t, res)
			},
		},
		{
			name:      "Single problem",
			classCode: "CS101",
			groupedProblems: []services.ProblemsByTopic{
				{
					Topic: "Arrays",
					Problems: []data.Problem{
						{
							ID:   1,
							Name: "Two Sum",
							URL:  "https://leetcode.com/problems/two-sum",
							Difficulty: ptrStr("Easy"),
						},
					},
				},
			},
			assertFunc: func(t *testing.T, res []*discordgo.MessageEmbed) {
				assert.Len(t, res, 1)
				assert.Equal(t, "List of problems for class CS101", res[0].Title)
				assert.Len(t, res[0].Fields, 3)
				assert.Contains(t, res[0].Fields[0].Value, "**Arrays**")
				assert.Contains(t, res[0].Fields[0].Value, "1")
				assert.Contains(t, res[0].Fields[1].Value, "🟩")
				assert.Contains(t, res[0].Fields[2].Value, "[Two Sum](https://leetcode.com/problems/two-sum)")
			},
		},
		{
			name:      "Multiple topics",
			classCode: "CS101",
			groupedProblems: []services.ProblemsByTopic{
				{
					Topic: "Arrays",
					Problems: []data.Problem{
						{ID: 1, Name: "Two Sum", URL: "url1", Difficulty: ptrStr("Easy")},
					},
				},
				{
					Topic: "Strings",
					Problems: []data.Problem{
						{ID: 2, Name: "Reverse", URL: "url2", Difficulty: ptrStr("Medium")},
					},
				},
			},
			assertFunc: func(t *testing.T, res []*discordgo.MessageEmbed) {
				assert.Len(t, res, 1)
				assert.Contains(t, res[0].Fields[0].Value, "**Arrays**")
				assert.Contains(t, res[0].Fields[0].Value, "**Strings**")
				assert.Contains(t, res[0].Fields[1].Value, "🟩")
				assert.Contains(t, res[0].Fields[1].Value, "🟨")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res := FormatProblemsAsEmbeds(tt.classCode, tt.groupedProblems)
			if tt.assertFunc != nil {
				tt.assertFunc(t, res)
			}
		})
	}
}

func TestGetDifficultyEmoji(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		difficulty data.Difficulty
		expected   string
	}{
		{"Easy", data.DifficultyEasy, "🟩"},
		{"Medium", data.DifficultyMedium, "🟨"},
		{"Hard", data.DifficultyHard, "🟥"},
		{"Default", data.DifficultyDefault, "⬜"},
		{"Unknown", data.Difficulty("Unknown"), "⬜"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, GetDifficultyEmoji(tt.difficulty))
		})
	}
}

func ptrStr(s string) *string { return &s }
