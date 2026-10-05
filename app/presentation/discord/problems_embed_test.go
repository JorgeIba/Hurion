package discord

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

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
							Difficulty: new("easy"),
							Priority:   new("recommended"),
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
				assert.Equal(t, "Difficulty / Priority", res[0].Fields[1].Name)
				assert.Contains(t, res[0].Fields[1].Value, "Recommended")
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
						{ID: 1, Name: "Two Sum", URL: "url1", Difficulty: new("easy")},
					},
				},
				{
					Topic: "Strings",
					Problems: []presentation.Problem{
						{ID: 2, Name: "Reverse", URL: "url2", Difficulty: new("medium")},
					},
				},
			},
			assertFunc: func(t *testing.T, res []*discordgo.MessageEmbed) {
				assert.Len(t, res, 1)
				assert.Contains(t, res[0].Fields[0].Value, "**Arrays**")
				assert.Contains(t, res[0].Fields[0].Value, "**Strings**")
				assert.Contains(t, res[0].Fields[1].Value, "🟩")
				assert.Contains(t, res[0].Fields[1].Value, "🟨")
				assert.Contains(t, res[0].Fields[1].Value, "No priority assigned")
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
						Difficulty: new("easy"),
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
							Difficulty: new("easy"),
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

func TestFormatProblemsAsEmbeds_OversizePriority(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		priority string
	}{
		{name: "ASCII", priority: strings.Repeat("A", 1500)},
		{name: "emoji", priority: strings.Repeat("😀", 750)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			problems := []presentation.ProblemsByTopic{{
				Topic: "Arrays",
				Problems: []presentation.Problem{
					{ID: 1, Name: "Before", URL: "url1"},
					{ID: 2, Name: "Oversized", URL: "url2", Difficulty: new("easy"), Priority: &tt.priority},
					{ID: 3, Name: "After", URL: "url3"},
				},
			}}
			embeds := FormatProblemsAsEmbeds("CS101", problems)
			if !assert.Len(t, embeds, 3) {
				return
			}
			for i, embed := range embeds {
				if !assert.Len(t, embed.Fields, 3) {
					continue
				}
				for _, field := range embed.Fields {
					assert.LessOrEqual(t, utf16Count(field.Value), maxFieldLength)
					assert.True(t, utf8.ValidString(field.Value))
				}
				assert.Equal(t, fmt.Sprintf("**Arrays**\n%d", i+1), embed.Fields[0].Value)
				assert.Equal(t, fmt.Sprintf("\u200b\n[%s](%s)", problems[0].Problems[i].Name, problems[0].Problems[i].URL), embed.Fields[2].Value)
			}
			metadata := embeds[1].Fields[1].Value
			assert.True(t, strings.HasPrefix(metadata, "\u200b\n🟩 / "))
			assert.True(t, strings.HasSuffix(metadata, "…"))
			assert.NotContains(t, metadata, "�")
			assert.Contains(t, embeds[0].Fields[1].Value, "No priority assigned")
			assert.Contains(t, embeds[2].Fields[1].Value, "No priority assigned")
		})
	}
}

func TestGetDifficultyEmoji(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		difficulty *string
		expected   string
	}{
		{"Easy", new("easy"), "🟩"},
		{"Medium", new("medium"), "🟨"},
		{"Hard", new("hard"), "🟥"},
		{"Default", nil, "⬜"},
		{"Unknown", new("Unknown"), "⬜"},
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
