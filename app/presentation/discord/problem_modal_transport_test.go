package discord_test

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/JorgeIba/Hurion/app/presentation"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

func TestBuildCreateProblemModalResponseEncodesLabelsPrefillAndRequiredInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		modal discord.CreateProblemModal
	}{
		{
			name: "unassigned metadata is prefilled",
			modal: discord.CreateProblemModal{
				ClassCode:          "basicos_1",
				ProblemModalFields: discord.ProblemModalFields{Difficulty: presentation.DifficultyUnspecifiedLabel, Priority: presentation.PriorityUnspecifiedLabel},
			},
		},
		{
			name: "all supplied text is preserved for prefilling",
			modal: discord.CreateProblemModal{
				ClassCode: "basicos_1",
				ProblemModalFields: discord.ProblemModalFields{
					Name: "Two Sum 🚀", URL: "https://example.com/problem?id=2",
					Difficulty: " Medium ", Priority: " Recommended ", Topic: "Binary search and two pointers",
				},
			},
		},
		{
			name: "oversized draft values are preserved for correction",
			modal: discord.CreateProblemModal{
				ClassCode: "basicos_1",
				ProblemModalFields: discord.ProblemModalFields{
					Name: strings.Repeat("🚀", 201), URL: "https://example.com/" + strings.Repeat("a", 2000),
					Topic: strings.Repeat("é", 101),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			response, err := discord.BuildCreateProblemModalResponse(tt.modal)
			if !assert.NoError(t, err) {
				return
			}
			encoded, err := json.Marshal(response)
			if !assert.NoError(t, err) {
				return
			}
			var wire struct {
				Type discordgo.InteractionResponseType `json:"type"`
				Data struct {
					CustomID   string `json:"custom_id"`
					Title      string `json:"title"`
					Components []struct {
						Type        discordgo.ComponentType `json:"type"`
						Label       string                  `json:"label"`
						Description string                  `json:"description"`
						Component   struct {
							Type      discordgo.ComponentType `json:"type"`
							CustomID  string                  `json:"custom_id"`
							Label     string                  `json:"label"`
							Value     string                  `json:"value"`
							Required  *bool                   `json:"required"`
							MaxLength *int                    `json:"max_length"`
						} `json:"component"`
					} `json:"components"`
				} `json:"data"`
			}
			if !assert.NoError(t, json.Unmarshal(encoded, &wire)) {
				return
			}
			assert.Equal(t, discordgo.InteractionResponseModal, wire.Type)
			assert.Equal(t, discord.CreateProblemModalCustomID+"|basicos_1", wire.Data.CustomID)
			assert.LessOrEqual(t, len(wire.Data.CustomID), 100)
			assert.Equal(t, "Add problem", wire.Data.Title)
			if !assert.Len(t, wire.Data.Components, 5) {
				return
			}
			expected := []struct {
				id, label, value string
				required         bool
			}{
				{"name", "Name", tt.modal.Name, true},
				{"url", "URL", tt.modal.URL, true},
				{"difficulty", "Difficulty", tt.modal.Difficulty, false},
				{"priority", "Priority", tt.modal.Priority, false},
				{"topic", "Topic", tt.modal.Topic, false},
			}
			for i, want := range expected {
				label := wire.Data.Components[i]
				assert.Nil(t, label.Component.MaxLength, "custom input limits must be omitted from the payload")
				assert.Equal(t, discordgo.LabelComponent, label.Type)
				assert.Equal(t, want.label, label.Label)
				switch want.id {
				case "name":
					assert.Equal(t, "Up to 200 characters", label.Description)
				case "url":
					assert.Equal(t, "HTTP or HTTPS; up to 2000 characters", label.Description)
				case "topic":
					assert.Equal(t, "Optional; up to 100 characters", label.Description)
				}
				assert.Equal(t, discordgo.TextInputComponent, label.Component.Type)
				assert.Equal(t, want.id, label.Component.CustomID)
				assert.Equal(t, want.value, label.Component.Value)
				assert.Empty(t, label.Component.Label, "the label belongs to its wrapper")
				if assert.NotNil(t, label.Component.Required, "required=false must reach Discord explicitly") {
					assert.Equal(t, want.required, *label.Component.Required)
				}
			}
		})
	}
}

func TestCreateProblemModalContextPreservesClassCodesThroughSDKSubmission(t *testing.T) {
	t.Parallel()
	for _, classCode := range []string{"basicos_1", "básicos_2026", "class|with%separators"} {
		t.Run(classCode, func(t *testing.T) {
			t.Parallel()
			response, err := discord.BuildCreateProblemModalResponse(discord.CreateProblemModal{ClassCode: classCode})
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, discord.CreateProblemModalCustomID, discord.ModalRouteFromCustomID(response.Data.CustomID))
			var event discordgo.InteractionCreate
			if !assert.NoError(t, json.Unmarshal([]byte(createProblemModalSubmission), &event)) {
				return
			}
			data := event.ModalSubmitData()
			data.CustomID = response.Data.CustomID
			modal, err := discord.ParseCreateProblemModal(data)
			assert.NoError(t, err)
			assert.Equal(t, classCode, modal.ClassCode)
		})
	}
}

func TestBuildCreateProblemModalResponseRejectsMissingClassAndEnforcesCustomIDLimit(t *testing.T) {
	t.Parallel()
	classCapacity := 100 - len(discord.CreateProblemModalCustomID) - len("|")
	response, err := discord.BuildCreateProblemModalResponse(discord.CreateProblemModal{ClassCode: strings.Repeat("a", classCapacity)})
	if assert.NoError(t, err) {
		assert.Len(t, response.Data.CustomID, 100)
	}
	for _, classCode := range []string{"", " \t ", strings.Repeat("a", classCapacity+1)} {
		response, err := discord.BuildCreateProblemModalResponse(discord.CreateProblemModal{ClassCode: classCode})
		assert.Error(t, err)
		assert.Nil(t, response)
	}
}

// Discord's submission payload has labels wrapping inputs, without the outgoing
// label, style, or required attributes. Decode it through the actual SDK first.
const createProblemModalSubmission = `{
	"type": 5,
	"guild_id": "guild_1",
	"data": {
		"custom_id": "huron:problem:create|basicos_1",
		"components": [
			{"type":18,"id":3,"component":{"type":4,"id":4,"custom_id":"name","value":"  Two Sum 🚀  "}},
			{"type":18,"id":5,"component":{"type":4,"id":6,"custom_id":"url","value":" https://example.com/problem?id=2 "}},
			{"type":18,"id":7,"component":{"type":4,"id":8,"custom_id":"difficulty","value":" mEDium "}},
			{"type":18,"id":11,"component":{"type":4,"id":12,"custom_id":"priority","value":" Recommended "}},
			{"type":18,"id":9,"component":{"type":4,"id":10,"custom_id":"topic","value":"Binary search\nTwo pointers"}}
		]
	}
}`

func TestParseCreateProblemModalReadsSDKSubmissionsByFieldIDAndPreservesRawText(t *testing.T) {
	t.Parallel()

	expected := discord.CreateProblemModal{
		ClassCode: "basicos_1",
		ProblemModalFields: discord.ProblemModalFields{
			Name: "  Two Sum 🚀  ", URL: " https://example.com/problem?id=2 ",
			Difficulty: " mEDium ", Priority: " Recommended ", Topic: "Binary search\nTwo pointers",
		},
	}
	blankOptional := expected
	blankOptional.Difficulty, blankOptional.Priority, blankOptional.Topic = "", "", ""
	invalidValues := expected
	invalidValues.Name, invalidValues.Difficulty, invalidValues.Priority = "", "Insane", "Urgent"

	tests := []struct {
		name   string
		change func(*discordgo.ModalSubmitInteractionData)
		want   discord.CreateProblemModal
	}{
		{name: "multiword unicode and whitespace survive the SDK and parser", want: expected},
		{
			name: "fields are identified by custom ID rather than their position", want: expected,
			change: func(data *discordgo.ModalSubmitInteractionData) {
				data.Components[0], data.Components[4] = data.Components[4], data.Components[0]
			},
		},
		{
			name: "empty optional inputs stay empty", want: blankOptional,
			change: func(data *discordgo.ModalSubmitInteractionData) {
				data.Components[2].(*discordgo.Label).Component.(*discordgo.TextInput).Value = ""
				data.Components[3].(*discordgo.Label).Component.(*discordgo.TextInput).Value = ""
				data.Components[4].(*discordgo.Label).Component.(*discordgo.TextInput).Value = ""
			},
		},
		{
			name: "domain-invalid values remain available for later validation", want: invalidValues,
			change: func(data *discordgo.ModalSubmitInteractionData) {
				data.Components[0].(*discordgo.Label).Component.(*discordgo.TextInput).Value = ""
				data.Components[2].(*discordgo.Label).Component.(*discordgo.TextInput).Value = "Insane"
				data.Components[3].(*discordgo.Label).Component.(*discordgo.TextInput).Value = "Urgent"
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var event discordgo.InteractionCreate
			if !assert.NoError(t, json.Unmarshal([]byte(createProblemModalSubmission), &event)) {
				return
			}
			data := event.ModalSubmitData()
			if tt.change != nil {
				tt.change(&data)
			}
			modal, err := discord.ParseCreateProblemModal(data)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, modal)
		})
	}
}

func TestParseCreateProblemModalRejectsIncompleteAmbiguousAndUnexpectedComponents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		change func(*discordgo.ModalSubmitInteractionData)
	}{
		{"different form ID", func(data *discordgo.ModalSubmitInteractionData) { data.CustomID = "other-form" }},
		{"missing class context", func(data *discordgo.ModalSubmitInteractionData) { data.CustomID = discord.CreateProblemModalCustomID }},
		{"invalid class encoding", func(data *discordgo.ModalSubmitInteractionData) {
			data.CustomID = discord.CreateProblemModalCustomID + "|%ZZ"
		}},
		{"empty class context", func(data *discordgo.ModalSubmitInteractionData) {
			data.CustomID = discord.CreateProblemModalCustomID + "|"
		}},
		{"missing input", func(data *discordgo.ModalSubmitInteractionData) { data.Components = data.Components[:4] }},
		{"repeated input ID", func(data *discordgo.ModalSubmitInteractionData) {
			data.Components[4] = data.Components[0]
		}},
		{"unknown input ID", func(data *discordgo.ModalSubmitInteractionData) {
			data.Components[0].(*discordgo.Label).Component.(*discordgo.TextInput).CustomID = "unknown"
		}},
		{"unexpected layout", func(data *discordgo.ModalSubmitInteractionData) {
			data.Components[0] = &discordgo.ActionsRow{}
		}},
		{"unexpected input type", func(data *discordgo.ModalSubmitInteractionData) {
			data.Components[0].(*discordgo.Label).Component = &discordgo.SelectMenu{}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var event discordgo.InteractionCreate
			if !assert.NoError(t, json.Unmarshal([]byte(createProblemModalSubmission), &event)) {
				return
			}
			data := event.ModalSubmitData()
			tt.change(&data)
			modal, err := discord.ParseCreateProblemModal(data)
			assert.Error(t, err)
			assert.Equal(t, discord.CreateProblemModal{}, modal, "a malformed form must not produce a partial request")
		})
	}
}

func TestFormatCreateProblemErrorPreservesDetailsWithinLimitAndFallsBackAboveIt(t *testing.T) {
	t.Parallel()
	fallback := "Could not add the problem. Please try /problem add again."
	for _, test := range []struct {
		name         string
		content      string
		want         string
		recoverDraft bool
	}{
		{name: "short error keeps correction details", content: "Please correct the values: URL must be an absolute HTTP or HTTPS URL. Try /problem add class:basicos_2 again.", want: "Please correct the values: URL must be an absolute HTTP or HTTPS URL. Try /problem add class:basicos_2 again."},
		{name: "exactly 2000 ASCII units are preserved", content: strings.Repeat("a", 2000), want: strings.Repeat("a", 2000)},
		{name: "2001 ASCII units use the selected class in the fallback", content: strings.Repeat("a", 2001), want: fallback},
		{name: "exactly 2000 UTF-16 units are preserved despite larger UTF-8 length", content: strings.Repeat("🚀", 1000), want: strings.Repeat("🚀", 1000)},
		{name: "2001 UTF-16 units use the fallback", content: strings.Repeat("🚀", 1000) + "a", want: fallback},
		{name: "recoverable error gets retry instruction", content: "Invalid URL", recoverDraft: true, want: "Invalid URL. Try /problem add class:basicos_2 again."},
		{name: "long recoverable error keeps selected class", content: strings.Repeat("a", 2001), recoverDraft: true, want: "Could not add the problem. Please check the values and try /problem add class:basicos_2 again."},
		{name: "retry instruction counts toward limit", content: strings.Repeat("a", 2000), recoverDraft: true, want: "Could not add the problem. Please check the values and try /problem add class:basicos_2 again."},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			content := discord.FormatCreateProblemError(test.content, "basicos_2", test.recoverDraft)
			assert.Equal(t, test.want, content)
			assert.LessOrEqual(t, len(utf16.Encode([]rune(content))), 2000)
		})
	}
}

func TestFormatCreatedProblemReceiptPreservesIdentityWithinDiscordMessageLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		assertFunc func(*testing.T, string)
	}{
		{
			name:  "normal text is shown",
			input: "Two Sum",
			assertFunc: func(t *testing.T, content string) {
				t.Helper()
				assert.Equal(t, "Problem `42` was added to class \"basicos_1\".\nName: \"Two Sum\"", content)
			},
		},
		{
			name:  "newlines stay inside their field",
			input: "Two\nSum",
			assertFunc: func(t *testing.T, content string) {
				t.Helper()
				assert.Equal(t, "Problem `42` was added to class \"basicos_1\".\nName: \"Two\\nSum\"", content)
			},
		},
		{
			name:  "unicode name within the UTF-16 limit is shown even when its UTF-8 bytes exceed 2000",
			input: strings.Repeat("🚀", 600),
			assertFunc: func(t *testing.T, content string) {
				t.Helper()
				assert.Equal(t, "Problem `42` was added to class \"basicos_1\".\nName: \""+strings.Repeat("🚀", 600)+"\"", content)
			},
		},
		{
			name:  "large unicode name is omitted from the receipt",
			input: strings.Repeat("🚀", 1000),
			assertFunc: func(t *testing.T, content string) {
				t.Helper()
				assert.Equal(t, "Problem `42` was added to class \"basicos_1\".", content)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			content := discord.FormatCreatedProblemReceipt(42, "basicos_1", tt.input)
			tt.assertFunc(t, content)
			assert.LessOrEqual(t, len(utf16.Encode([]rune(content))), 2000)
		})
	}
}
