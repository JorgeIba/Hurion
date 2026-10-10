package interactiontransport_test

import (
	"encoding/json"
	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"strconv"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func editSubmission(t *testing.T, response *discordgo.InteractionResponse) discordgo.ModalSubmitInteractionData {
	t.Helper()
	data := discordgo.ModalSubmitInteractionData{CustomID: response.Data.CustomID}
	for _, component := range response.Data.Components {
		label, ok := component.(discordgo.Label)
		require.True(t, ok, "expected an outgoing label, got %T", component)
		submittedLabel := &discordgo.Label{}
		switch field := label.Component.(type) {
		case discordgo.TextInput:
			submittedLabel.Component = &discordgo.TextInput{CustomID: field.CustomID, Value: field.Value}
		default:
			encoded, err := json.Marshal(label.Component)
			require.NoError(t, err)
			var wireField struct {
				CustomID string `json:"custom_id"`
				Options  []struct {
					Value   string `json:"value"`
					Default bool   `json:"default"`
				} `json:"options"`
			}
			require.NoError(t, json.Unmarshal(encoded, &wireField))
			selection := &discordgo.SelectMenu{CustomID: wireField.CustomID}
			for _, option := range wireField.Options {
				if option.Default {
					selection.Values = []string{option.Value}
					break
				}
			}
			submittedLabel.Component = selection
		}
		data.Components = append(data.Components, submittedLabel)
	}
	return data
}

func TestProblemModalBuildersEnforceSharedFieldLimit(t *testing.T) {
	t.Parallel()
	for _, field := range []string{"name", "url", "topic"} {
		for _, value := range []string{strings.Repeat("🚀", 2000), strings.Repeat("🚀", 2000) + "a"} {
			t.Run(field+"/"+strconv.Itoa(countUTF16ForTest(value)), func(t *testing.T) {
				fields := interactiontransport.ProblemModalFields{}
				switch field {
				case "name":
					fields.Name = value
				case "url":
					fields.URL = value
				case "topic":
					fields.Topic = &value
				}
				for _, build := range []func() (*discordgo.InteractionResponse, error){
					func() (*discordgo.InteractionResponse, error) {
						return buildCreateProblemModalForTest(interactiontransport.CreateProblemModal{ClassCode: "basicos", ProblemModalFields: fields})
					},
					func() (*discordgo.InteractionResponse, error) {
						return buildEditProblemModalForTest(interactiontransport.EditProblemModal{ProblemID: 123, ClassCode: "basicos", ProblemModalFields: fields})
					},
				} {
					response, err := build()
					if countUTF16ForTest(value) > 4000 {
						require.ErrorContains(t, err, "field exceeds")
						assert.Nil(t, response)
					} else {
						require.NoError(t, err)
						assert.NotNil(t, response)
					}
				}
			})
		}
	}
}

func TestEditProblemModalRejectsMalformedSubmissions(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		mutate func(*discordgo.ModalSubmitInteractionData)
	}{
		{"missing context", func(d *discordgo.ModalSubmitInteractionData) { d.CustomID = string(discordcommands.EditProblem) }},
		{"missing class", func(d *discordgo.ModalSubmitInteractionData) {
			d.CustomID = string(discordcommands.EditProblem) + "|123"
		}},
		{"zero problem", func(d *discordgo.ModalSubmitInteractionData) {
			d.CustomID = string(discordcommands.EditProblem) + "|basicos|0"
		}},
		{"negative problem", func(d *discordgo.ModalSubmitInteractionData) {
			d.CustomID = string(discordcommands.EditProblem) + "|basicos|-1"
		}},
		{"invalid problem", func(d *discordgo.ModalSubmitInteractionData) {
			d.CustomID = string(discordcommands.EditProblem) + "|basicos|abc"
		}},
		{"blank class", func(d *discordgo.ModalSubmitInteractionData) {
			d.CustomID = string(discordcommands.EditProblem) + "||123"
		}},
		{"invalid class escape", func(d *discordgo.ModalSubmitInteractionData) {
			d.CustomID = string(discordcommands.EditProblem) + "|%GG|123"
		}},
		{"overflow problem", func(d *discordgo.ModalSubmitInteractionData) {
			d.CustomID = string(discordcommands.EditProblem) + "|basicos|9223372036854775808"
		}},
		{"extra context", func(d *discordgo.ModalSubmitInteractionData) { d.CustomID += "|123" }},
		{"missing field", func(d *discordgo.ModalSubmitInteractionData) { d.Components = d.Components[:4] }},
		{"repeated field", func(d *discordgo.ModalSubmitInteractionData) { d.Components[4] = d.Components[0] }},
		{"unexpected field", func(d *discordgo.ModalSubmitInteractionData) {
			d.Components[0].(*discordgo.Label).Component.(*discordgo.TextInput).CustomID = "problem_id"
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			response, err := buildEditProblemModalForTest(interactiontransport.EditProblemModal{ProblemID: 123, ClassCode: "basicos"})
			require.NoError(t, err)
			data := editSubmission(t, response)
			test.mutate(&data)
			modal, err := interactiontransport.ParseEditProblemModal(data)
			require.Error(t, err)
			assert.Equal(t, interactiontransport.EditProblemModal{}, modal)
		})
	}
}

func TestEditProblemModalCustomIDLimits(t *testing.T) {
	t.Parallel()
	capacity := 100 - len(string(discordcommands.EditProblem)) - len("||123")
	modal := interactiontransport.EditProblemModal{ProblemID: 123, ClassCode: strings.Repeat("a", capacity)}
	response, err := buildEditProblemModalForTest(modal)
	require.NoError(t, err)
	assert.Len(t, response.Data.CustomID, 100)
	assert.LessOrEqual(t, countUTF16ForTest(response.Data.Title), 45)
	parsed, err := interactiontransport.ParseEditProblemModal(editSubmission(t, response))
	require.NoError(t, err)
	assert.Equal(t, modal.ClassCode, parsed.ClassCode)
	for _, classCode := range []string{"", " ", strings.Repeat("a", capacity+1), strings.Repeat("🚀", capacity)} {
		modal.ClassCode = classCode
		response, err := buildEditProblemModalForTest(modal)
		require.Error(t, err)
		assert.Nil(t, response)
	}
}

func TestEditProblemModalLegacyValuesHaveRepairPath(t *testing.T) {
	t.Parallel()
	for _, test := range []struct{ field string }{{"name"}, {"url"}, {"topic"}} {
		t.Run(test.field, func(t *testing.T) {
			t.Parallel()
			modal := interactiontransport.EditProblemModal{ProblemID: 123, ClassCode: "basicos"}
			value := strings.Repeat("🚀", 2001)
			switch test.field {
			case "name":
				modal.Name = value
			case "url":
				modal.URL = value
			case "topic":
				modal.Topic = &value
			}
			response, err := buildEditProblemModalForTest(modal)
			require.ErrorContains(t, err, "!update_problem")
			assert.Nil(t, response, "legacy fields must not be silently truncated")
		})
	}
}

func countUTF16ForTest(value string) int { return len(utf16.Encode([]rune(value))) }

func TestEditProblemModalRoundTripAndDisplayLimits(t *testing.T) {
	t.Parallel()
	raw := interactiontransport.ProblemModalFields{Name: " Raw 🚀 ", URL: " invalid ", Difficulty: new(" HARD "), Priority: new("recommended"), Topic: new(" Graphs ")}
	response, err := buildEditProblemModalForTest(interactiontransport.EditProblemModal{ProblemID: 123, ClassCode: "basicos|á%", ProblemModalFields: raw})
	require.NoError(t, err)
	assert.LessOrEqual(t, countUTF16ForTest(response.Data.Title), 45)
	assert.True(t, strings.HasPrefix(response.Data.Title, "Edit #123"))
	require.Len(t, response.Data.Components, 5)
	modal, err := interactiontransport.ParseEditProblemModal(editSubmission(t, response))
	require.NoError(t, err)
	assert.EqualValues(t, 123, modal.ProblemID)
	assert.Equal(t, "basicos|á%", modal.ClassCode)
	expected := raw
	expected.Difficulty, expected.Priority = new("hard"), new("recommended")
	assert.Equal(t, expected, modal.ProblemModalFields, "text is preserved and selections return canonical codes")
}
