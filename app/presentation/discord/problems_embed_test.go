package discord

import (
	"fmt"
	"strings"
	"testing"

	"github.com/JorgeIba/Hurion/app/presentation"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

func TestFormatProblemsAsEmbeds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		classCode       string
		groupedProblems []presentation.ProblemsByTopic
		assertFunc      func(t *testing.T, res []*discordgo.MessageEmbed)
	}{
		{
			name:            "Empty problems",
			classCode:       "CS101",
			groupedProblems: []presentation.ProblemsByTopic{},
			assertFunc: func(t *testing.T, res []*discordgo.MessageEmbed) {
				assert.Empty(t, res)
			},
		},
		{
			name:      "Single problem",
			classCode: "CS101",
			groupedProblems: []presentation.ProblemsByTopic{
				{
					Topic: "Arrays",
					Problems: []presentation.Problem{
						{
							ID:         1,
							Name:       "Two Sum",
							URL:        "https://leetcode.com/problems/two-sum",
							Difficulty: "Easy",
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
			groupedProblems: []presentation.ProblemsByTopic{
				{
					Topic: "Arrays",
					Problems: []presentation.Problem{
						{ID: 1, Name: "Two Sum", URL: "url1", Difficulty: "Easy"},
					},
				},
				{
					Topic: "Strings",
					Problems: []presentation.Problem{
						{ID: 2, Name: "Reverse", URL: "url2", Difficulty: "Medium"},
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
		{
			name:      "Pagination - Exceeds 1024 chars",
			classCode: "CS101",
			groupedProblems: func() []presentation.ProblemsByTopic {
				var problems []presentation.Problem
				// Create enough problems to force pagination.
				for i := 1; i <= 60; i++ {
					problems = append(problems, presentation.Problem{
						ID:         int64(i),
						Name:       fmt.Sprintf("Problem %d", i),
						URL:        "http://url",
						Difficulty: "Easy",
					})
				}
				return []presentation.ProblemsByTopic{
					{
						Topic:    "Many Problems",
						Problems: problems,
					},
				}
			}(),
			assertFunc: func(t *testing.T, res []*discordgo.MessageEmbed) {
				assert.GreaterOrEqual(t, len(res), 2, "Should split into multiple embeds")
				for _, embed := range res {
					assert.Len(t, embed.Fields, 3)
					for _, field := range embed.Fields {
						assert.LessOrEqual(t, utf16Count(field.Value), maxFieldLength, "Field value should not exceed max length (UTF-16 code units)")
					}
				}
			},
		},
		{
			name:      "Oversize single row truncation",
			classCode: "CS101",
			groupedProblems: []presentation.ProblemsByTopic{
				{
					Topic: "Big Problem",
					Problems: []presentation.Problem{
						{
							ID:         1,
							Name:       strings.Repeat("A", 1500), // Larger than maxFieldLength
							URL:        "http://url",
							Difficulty: "Easy",
						},
					},
				},
			},
			assertFunc: func(t *testing.T, res []*discordgo.MessageEmbed) {
				assert.Len(t, res, 1)
				assert.Len(t, res[0].Fields, 3)
				for _, field := range res[0].Fields {
					assert.LessOrEqual(t, utf16Count(field.Value), maxFieldLength, "Field value should not exceed max length even for oversized single problem")
				}
				assert.Contains(t, res[0].Fields[2].Value, "...", "The name should have been truncated")
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
		difficulty string
		expected   string
	}{
		{"Easy", "Easy", "🟩"},
		{"Medium", "Medium", "🟨"},
		{"Hard", "Hard", "🟥"},
		{"Default", "Not specified", "⬜"},
		{"Unknown", "Unknown", "⬜"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, GetDifficultyEmoji(tt.difficulty))
		})
	}
}

func TestUtf16Count(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected int
	}{
		{"abc", 3},
		{"😊", 2}, // Emoji outside BMP uses 2 code units
		{"a😊b", 4},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, utf16Count(tt.input))
		})
	}
}
