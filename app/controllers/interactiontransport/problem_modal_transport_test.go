package interactiontransport_test

import (
	"encoding/json"
	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

func TestBuildCreateProblemModalResponseEncodesLabelsPrefillAndRequiredInputs(t *testing.T) {
	t.Parallel()
	type wireOption struct {
		Label   string `json:"label"`
		Value   string `json:"value"`
		Default bool   `json:"default"`
	}
	type wireComponent struct {
		Type      discordgo.ComponentType `json:"type"`
		CustomID  string                  `json:"custom_id"`
		Value     string                  `json:"value"`
		Required  *bool                   `json:"required"`
		MaxLength *int                    `json:"max_length"`
		Disabled  *bool                   `json:"disabled"`
		Options   []wireOption            `json:"options"`
	}
	type wireLabel struct {
		Type        discordgo.ComponentType `json:"type"`
		Label       string                  `json:"label"`
		Description string                  `json:"description"`
		Component   wireComponent           `json:"component"`
	}

	tests := []struct {
		name               string
		modal              interactiontransport.CreateProblemModal
		selectedDifficulty string
		selectedPriority   string
	}{
		{
			name: "unassigned metadata is prefilled",
			modal: interactiontransport.CreateProblemModal{
				ClassCode:          "basicos_1",
				ProblemModalFields: interactiontransport.ProblemModalFields{},
			},
			selectedDifficulty: "unset",
			selectedPriority:   "unset",
		},
		{
			name: "text is preserved and metadata codes preselect options",
			modal: interactiontransport.CreateProblemModal{
				ClassCode: "basicos_1",
				ProblemModalFields: interactiontransport.ProblemModalFields{
					Name: "Two Sum 🚀", URL: "https://example.com/problem?id=2",
					Difficulty: new(" Medium "), Priority: new("recommended"), Topic: new("Binary search and two pointers"),
				},
			},
			selectedDifficulty: "medium",
			selectedPriority:   "recommended",
		},
		{
			name: "oversized draft values are preserved for correction",
			modal: interactiontransport.CreateProblemModal{
				ClassCode: "basicos_1",
				ProblemModalFields: interactiontransport.ProblemModalFields{
					Name: strings.Repeat("🚀", 201), URL: "https://example.com/" + strings.Repeat("a", 2000),
					Topic: new(strings.Repeat("é", 101)),
				},
			},
			selectedDifficulty: "unset",
			selectedPriority:   "unset",
		},
		{
			name: "unknown metadata does not add options or select a default",
			modal: interactiontransport.CreateProblemModal{ClassCode: "basicos_1", ProblemModalFields: interactiontransport.ProblemModalFields{
				Difficulty: new("extreme"), Priority: new(strings.Repeat("🚀", 1500)),
			}},
		},
		{
			name: "canonical codes preselect options",
			modal: interactiontransport.CreateProblemModal{ClassCode: "basicos_1", ProblemModalFields: interactiontransport.ProblemModalFields{
				Difficulty: new("hard"), Priority: new("not_required"),
			}},
			selectedDifficulty: "hard",
			selectedPriority:   "not_required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			response, err := buildCreateProblemModalForTest(tt.modal)
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
					CustomID   string      `json:"custom_id"`
					Title      string      `json:"title"`
					Components []wireLabel `json:"components"`
				} `json:"data"`
			}
			if !assert.NoError(t, json.Unmarshal(encoded, &wire)) {
				return
			}
			assert.Equal(t, discordgo.InteractionResponseModal, wire.Type)
			assert.Equal(t, string(discordcommands.CreateProblem)+"|basicos_1", wire.Data.CustomID)
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
				{"difficulty", "Difficulty", "", false},
				{"priority", "Priority", "", false},
				{"topic", "Topic", optionalProblemFormText(tt.modal.Topic), false},
			}
			difficultyOptions := []wireOption{
				{Label: "Easy", Value: "easy"},
				{Label: "Medium", Value: "medium"},
				{Label: "Hard", Value: "hard"},
				{Label: "Not specified", Value: "unset"},
			}
			for i := range difficultyOptions {
				difficultyOptions[i].Default = difficultyOptions[i].Value == tt.selectedDifficulty
			}
			priorityOptions := []wireOption{
				{Label: "Essential", Value: "essential"},
				{Label: "Recommended", Value: "recommended"},
				{Label: "Supplementary", Value: "supplementary"},
				{Label: "Not required", Value: "not_required"},
				{Label: "Discouraged", Value: "discouraged"},
				{Label: "No priority assigned", Value: "unset"},
			}
			for i := range priorityOptions {
				priorityOptions[i].Default = priorityOptions[i].Value == tt.selectedPriority
			}
			for i, want := range expected {
				label := wire.Data.Components[i]
				assert.Nil(t, label.Component.MaxLength, "custom input limits must be omitted from the payload")
				assert.Nil(t, label.Component.Disabled, "modal selects must omit Discord's unsupported disabled field")
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
				assert.Equal(t, want.id, label.Component.CustomID)
				if assert.NotNil(t, label.Component.Required, "required=false must reach Discord explicitly") {
					assert.Equal(t, want.required, *label.Component.Required)
				}
				switch want.id {
				case "name", "url", "topic":
					assert.Equal(t, discordgo.TextInputComponent, label.Component.Type)
					assert.Equal(t, want.value, label.Component.Value)
					assert.Empty(t, label.Component.Options)
				case "difficulty":
					assert.Equal(t, discordgo.SelectMenuComponent, label.Component.Type)
					assert.Empty(t, label.Component.Value)
					assert.Equal(t, difficultyOptions, label.Component.Options)
				case "priority":
					assert.Equal(t, discordgo.SelectMenuComponent, label.Component.Type)
					assert.Empty(t, label.Component.Value)
					assert.Equal(t, priorityOptions, label.Component.Options)
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
			response, err := buildCreateProblemModalForTest(interactiontransport.CreateProblemModal{ClassCode: classCode})
			if !assert.NoError(t, err) {
				return
			}
			route, _, _ := strings.Cut(response.Data.CustomID, "|")
			assert.Equal(t, string(discordcommands.CreateProblem), route)
			var event discordgo.InteractionCreate
			if !assert.NoError(t, json.Unmarshal([]byte(createProblemModalSubmission), &event)) {
				return
			}
			data := event.ModalSubmitData()
			data.CustomID = response.Data.CustomID
			modal, err := interactiontransport.ParseCreateProblemModal(data)
			assert.NoError(t, err)
			assert.Equal(t, classCode, modal.ClassCode)
		})
	}
}

func TestBuildCreateProblemModalResponseRejectsMissingClassAndEnforcesCustomIDLimit(t *testing.T) {
	t.Parallel()
	classCapacity := 100 - len(string(discordcommands.CreateProblem)) - len("|")
	response, err := buildCreateProblemModalForTest(interactiontransport.CreateProblemModal{ClassCode: strings.Repeat("a", classCapacity)})
	if assert.NoError(t, err) {
		assert.Len(t, response.Data.CustomID, 100)
	}
	for _, classCode := range []string{"", " \t ", strings.Repeat("a", classCapacity+1)} {
		response, err := buildCreateProblemModalForTest(interactiontransport.CreateProblemModal{ClassCode: classCode})
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
			{"type":18,"id":7,"component":{"type":3,"id":8,"custom_id":"difficulty","values":["medium"]}},
			{"type":18,"id":11,"component":{"type":3,"id":12,"custom_id":"priority","values":["recommended"]}},
			{"type":18,"id":9,"component":{"type":4,"id":10,"custom_id":"topic","value":"Binary search\nTwo pointers"}}
		]
	}
}`

func TestParseCreateProblemModalReadsSDKSubmissionsByFieldIDAndPreservesRawText(t *testing.T) {
	t.Parallel()

	expected := interactiontransport.CreateProblemModal{
		ClassCode: "basicos_1",
		ProblemModalFields: interactiontransport.ProblemModalFields{
			Name: "  Two Sum 🚀  ", URL: " https://example.com/problem?id=2 ",
			Difficulty: new("medium"), Priority: new("recommended"), Topic: new("Binary search\nTwo pointers"),
		},
	}
	blankOptional := expected
	blankOptional.Difficulty, blankOptional.Priority, blankOptional.Topic = nil, nil, nil
	blankMetadata := expected
	blankMetadata.Difficulty, blankMetadata.Priority = nil, nil
	invalidValues := expected
	invalidValues.Name, invalidValues.Difficulty, invalidValues.Priority = "", nil, nil

	tests := []struct {
		name   string
		change func(*discordgo.ModalSubmitInteractionData)
		want   interactiontransport.CreateProblemModal
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
				data.Components[2].(*discordgo.Label).Component.(*discordgo.SelectMenu).Values = nil
				data.Components[3].(*discordgo.Label).Component.(*discordgo.SelectMenu).Values = nil
				data.Components[4].(*discordgo.Label).Component.(*discordgo.TextInput).Value = ""
			},
		},
		{
			name: "unassigned selections become nil metadata", want: blankMetadata,
			change: func(data *discordgo.ModalSubmitInteractionData) {
				data.Components[2].(*discordgo.Label).Component.(*discordgo.SelectMenu).Values = []string{"unset"}
				data.Components[3].(*discordgo.Label).Component.(*discordgo.SelectMenu).Values = []string{"unset"}
			},
		},
		{
			name: "unknown selections are ignored while raw text reaches service validation", want: invalidValues,
			change: func(data *discordgo.ModalSubmitInteractionData) {
				data.Components[0].(*discordgo.Label).Component.(*discordgo.TextInput).Value = ""
				data.Components[2].(*discordgo.Label).Component.(*discordgo.SelectMenu).Values = []string{"Insane"}
				data.Components[3].(*discordgo.Label).Component.(*discordgo.SelectMenu).Values = []string{"Urgent"}
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
			modal, err := interactiontransport.ParseCreateProblemModal(data)
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
		{"missing class context", func(data *discordgo.ModalSubmitInteractionData) {
			data.CustomID = string(discordcommands.CreateProblem)
		}},
		{"invalid class encoding", func(data *discordgo.ModalSubmitInteractionData) {
			data.CustomID = string(discordcommands.CreateProblem) + "|%ZZ"
		}},
		{"empty class context", func(data *discordgo.ModalSubmitInteractionData) {
			data.CustomID = string(discordcommands.CreateProblem) + "|"
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
		{"multiple select values", func(data *discordgo.ModalSubmitInteractionData) {
			data.Components[2].(*discordgo.Label).Component.(*discordgo.SelectMenu).Values = []string{"easy", "hard"}
		}},
		{"non-string select type", func(data *discordgo.ModalSubmitInteractionData) {
			data.Components[2].(*discordgo.Label).Component.(*discordgo.SelectMenu).MenuType = discordgo.UserSelectMenu
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
			modal, err := interactiontransport.ParseCreateProblemModal(data)
			assert.Error(t, err)
			assert.Equal(t, interactiontransport.CreateProblemModal{}, modal, "a malformed form must not produce a partial request")
		})
	}
}

func optionalProblemFormText(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
