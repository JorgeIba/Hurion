package discord

import (
	"strconv"
	"strings"
	"testing"

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
		case modalStringSelect:
			selection := &discordgo.SelectMenu{CustomID: field.CustomID}
			for _, option := range field.Options {
				if option.Default {
					selection.Values = []string{option.Value}
					break
				}
			}
			submittedLabel.Component = selection
		default:
			t.Fatalf("unexpected form component %T", field)
		}
		data.Components = append(data.Components, submittedLabel)
	}
	return data
}

func TestProblemModalBuildersEnforceSharedFieldLimit(t *testing.T) {
	t.Parallel()
	for _, field := range []string{"name", "url", "topic"} {
		for _, value := range []string{strings.Repeat("🚀", 2000), strings.Repeat("🚀", 2000) + "a"} {
			t.Run(field+"/"+strconv.Itoa(utf16Count(value)), func(t *testing.T) {
				fields := ProblemModalFields{}
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
						return BuildCreateProblemModalResponse(CreateProblemModal{ClassCode: "basicos", ProblemModalFields: fields})
					},
					func() (*discordgo.InteractionResponse, error) {
						return BuildEditProblemModalResponse(EditProblemModal{ProblemID: 123, ClassCode: "basicos", ProblemModalFields: fields})
					},
				} {
					response, err := build()
					if utf16Count(value) > 4000 {
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

func TestEditProblemModalRoundTripAndDisplayLimits(t *testing.T) {
	t.Parallel()
	raw := ProblemModalFields{Name: " Raw 🚀 ", URL: " invalid ", Difficulty: new(" HARD "), Priority: new("recommended"), Topic: new(" Graphs ")}
	response, err := BuildEditProblemModalResponse(EditProblemModal{ProblemID: 123, ClassCode: "basicos|á%", ProblemModalFields: raw})
	require.NoError(t, err)
	assert.LessOrEqual(t, utf16Count(response.Data.Title), 45)
	assert.True(t, strings.HasPrefix(response.Data.Title, "Edit #123"))
	require.Len(t, response.Data.Components, 5)
	modal, err := ParseEditProblemModal(editSubmission(t, response))
	require.NoError(t, err)
	assert.EqualValues(t, 123, modal.ProblemID)
	assert.Equal(t, "basicos|á%", modal.ClassCode)
	expected := raw
	expected.Difficulty, expected.Priority = new("hard"), new("recommended")
	assert.Equal(t, expected, modal.ProblemModalFields, "text is preserved and selections return canonical codes")
	receipt := FormatUpdatedProblemReceipt(123, strings.Repeat("🚀", 2000), raw.Name)
	assert.LessOrEqual(t, utf16Count(receipt), 2000)
	assert.Contains(t, receipt, "123")
	assert.LessOrEqual(t, utf16Count(FormatEditProblemError(strings.Repeat("🚀", 2000), 123, "basicos", true)), 2000)
}

func TestEditProblemErrorPreservesDestinationWhenShortened(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		content   string
		classCode string
		want      string
	}{
		{"short error", "Invalid URL.", "avanzados", "Invalid URL. Try /problem edit problem:123 class:avanzados again."},
		{"long error", strings.Repeat("🚀", 2000), "avanzados", "class:avanzados"},
		{"class cannot fit", strings.Repeat("🚀", 2000), strings.Repeat("a", 2000), "same destination class code"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			content := FormatEditProblemError(test.content, 123, test.classCode, true)
			assert.Contains(t, content, test.want)
			assert.LessOrEqual(t, utf16Count(content), 2000)
		})
	}
}

func TestEditProblemErrorWithoutRecoverableDraft(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Internal error.", FormatEditProblemError("Internal error.", 123, "avanzados", false))
	content := FormatEditProblemError(strings.Repeat("🚀", 2000), 123, "avanzados", false)
	assert.Equal(t, "Could not update problem `123`. Please try /problem edit again.", content)
	assert.LessOrEqual(t, utf16Count(content), 2000)
}

func TestEditProblemModalRejectsMalformedSubmissions(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		mutate func(*discordgo.ModalSubmitInteractionData)
	}{
		{"missing context", func(d *discordgo.ModalSubmitInteractionData) { d.CustomID = EditProblemModalCustomID }},
		{"missing class", func(d *discordgo.ModalSubmitInteractionData) { d.CustomID = EditProblemModalCustomID + "|123" }},
		{"zero problem", func(d *discordgo.ModalSubmitInteractionData) { d.CustomID = EditProblemModalCustomID + "|basicos|0" }},
		{"negative problem", func(d *discordgo.ModalSubmitInteractionData) { d.CustomID = EditProblemModalCustomID + "|basicos|-1" }},
		{"invalid problem", func(d *discordgo.ModalSubmitInteractionData) { d.CustomID = EditProblemModalCustomID + "|basicos|abc" }},
		{"blank class", func(d *discordgo.ModalSubmitInteractionData) { d.CustomID = EditProblemModalCustomID + "||123" }},
		{"invalid class escape", func(d *discordgo.ModalSubmitInteractionData) { d.CustomID = EditProblemModalCustomID + "|%GG|123" }},
		{"overflow problem", func(d *discordgo.ModalSubmitInteractionData) {
			d.CustomID = EditProblemModalCustomID + "|basicos|9223372036854775808"
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
			response, err := BuildEditProblemModalResponse(EditProblemModal{ProblemID: 123, ClassCode: "basicos"})
			require.NoError(t, err)
			data := editSubmission(t, response)
			test.mutate(&data)
			modal, err := ParseEditProblemModal(data)
			require.Error(t, err)
			assert.Equal(t, EditProblemModal{}, modal)
		})
	}
}

func TestEditProblemModalCustomIDLimits(t *testing.T) {
	t.Parallel()
	capacity := modalCustomIDMaxLength - len(EditProblemModalCustomID) - len("||123")
	modal := EditProblemModal{ProblemID: 123, ClassCode: strings.Repeat("a", capacity)}
	response, err := BuildEditProblemModalResponse(modal)
	require.NoError(t, err)
	assert.Len(t, response.Data.CustomID, modalCustomIDMaxLength)
	assert.LessOrEqual(t, utf16Count(response.Data.Title), 45)
	parsed, err := ParseEditProblemModal(editSubmission(t, response))
	require.NoError(t, err)
	assert.Equal(t, modal.ClassCode, parsed.ClassCode)
	for _, classCode := range []string{"", " ", strings.Repeat("a", capacity+1), strings.Repeat("🚀", capacity)} {
		modal.ClassCode = classCode
		response, err := BuildEditProblemModalResponse(modal)
		require.Error(t, err)
		assert.Nil(t, response)
	}
}

func TestEditProblemModalLegacyValuesHaveRepairPath(t *testing.T) {
	t.Parallel()
	for _, test := range []struct{ field string }{{"name"}, {"url"}, {"topic"}} {
		t.Run(test.field, func(t *testing.T) {
			t.Parallel()
			modal := EditProblemModal{ProblemID: 123, ClassCode: "basicos"}
			value := strings.Repeat("🚀", 2001)
			switch test.field {
			case "name":
				modal.Name = value
			case "url":
				modal.URL = value
			case "topic":
				modal.Topic = &value
			}
			response, err := BuildEditProblemModalResponse(modal)
			require.ErrorContains(t, err, "!update_problem")
			assert.Nil(t, response, "legacy fields must not be silently truncated")
		})
	}
}
