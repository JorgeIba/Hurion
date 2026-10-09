package discord

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

func TestJSONModalEncoding(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		response *discordgo.InteractionResponse
		route    string
	}{
		{"problems", BuildProblemBatchModalResponse(), ProblemBatchModalCustomID},
		{"ACs", BuildACBatchModalResponse(), ACBatchModalCustomID},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			encoded, err := json.Marshal(test.response.Data)
			assert.NoError(t, err)
			var data discordgo.ModalSubmitInteractionData
			assert.NoError(t, json.Unmarshal(encoded, &data))
			assert.Equal(t, test.route, data.CustomID)
			if !assert.Len(t, data.Components, 2) {
				return
			}
			text := data.Components[0].(*discordgo.Label).Component.(*discordgo.TextInput)
			assert.Equal(t, discordgo.TextInputParagraph, text.Style)
			assert.Equal(t, 4000, text.MaxLength)
			assert.False(t, *text.Required)
			assert.Contains(t, text.Placeholder, "[{\"")
			upload := data.Components[1].(*discordgo.Label).Component.(*discordgo.FileUpload)
			assert.False(t, *upload.Required)
			assert.Zero(t, *upload.MinValues)
			assert.Equal(t, 1, upload.MaxValues)
		})
	}
}

func TestParseProblemBatchModalSource(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, text, filename string
		files                []string
		mutate               func(*discordgo.ModalSubmitInteractionData)
		wantError            string
	}{
		{name: "text preserved", text: " \n[] "},
		{name: "file", files: []string{"42"}, filename: "batch.json"},
		{name: "uppercase extension and whitespace text", text: " \n", files: []string{"42"}, filename: "batch.JSON"},
		{name: "neither source", wantError: "paste JSON"},
		{name: "file takes precedence", text: "not JSON", files: []string{"42"}, filename: "batch.json"},
		{name: "multiple files", files: []string{"42", "43"}, wantError: "only one"},
		{name: "extension checked by file reader", files: []string{"42"}, filename: "batch.txt"},
		{name: "ignored oversized text", text: strings.Repeat("🚀", 2001), files: []string{"42"}, filename: "batch.json"},
		{name: "missing attachment", files: []string{"42"}, mutate: func(d *discordgo.ModalSubmitInteractionData) { d.Resolved.Attachments = nil }, wantError: "resolve"},
		{name: "mismatched attachment", files: []string{"42"}, mutate: func(d *discordgo.ModalSubmitInteractionData) { d.Resolved.Attachments["42"].ID = "43" }, wantError: "resolve"},
		{name: "oversized text", text: strings.Repeat("🚀", 2001), wantError: "4000-character"},
		{name: "unknown field", text: "[]", mutate: func(d *discordgo.ModalSubmitInteractionData) {
			d.Components[0].(*discordgo.Label).Component.(*discordgo.TextInput).CustomID = "other"
		}, wantError: "unknown"},
		{name: "duplicate text", text: "[]", mutate: func(d *discordgo.ModalSubmitInteractionData) { d.Components = append(d.Components, d.Components[0]) }, wantError: "repeated"},
		{name: "duplicate file", text: "[]", mutate: func(d *discordgo.ModalSubmitInteractionData) { d.Components = append(d.Components, d.Components[1]) }, wantError: "repeated"},
		{name: "missing field", text: "[]", mutate: func(d *discordgo.ModalSubmitInteractionData) { d.Components = d.Components[:1] }, wantError: "incomplete"},
		{name: "wrong component", text: "[]", mutate: func(d *discordgo.ModalSubmitInteractionData) { d.Components[0] = &discordgo.ActionsRow{} }, wantError: "label"},
		{name: "nil label", text: "[]", mutate: func(d *discordgo.ModalSubmitInteractionData) { d.Components[0] = (*discordgo.Label)(nil) }, wantError: "label"},
		{name: "nil input", text: "[]", mutate: func(d *discordgo.ModalSubmitInteractionData) {
			d.Components[0].(*discordgo.Label).Component = (*discordgo.TextInput)(nil)
		}, wantError: "unknown"},
		{name: "wrong route", text: "[]", mutate: func(d *discordgo.ModalSubmitInteractionData) { d.CustomID = "other" }, wantError: "form ID"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			// Decode the actual wire shape rather than hand-constructing SDK labels.
			raw, err := json.Marshal(map[string]any{
				"custom_id": ProblemBatchModalCustomID,
				"components": []any{
					map[string]any{"type": 18, "component": map[string]any{"type": 4, "custom_id": "json", "value": test.text}},
					map[string]any{"type": 18, "component": map[string]any{"type": 19, "custom_id": "file", "values": test.files}},
				},
				"resolved": map[string]any{"attachments": map[string]any{"42": map[string]any{"id": "42", "filename": test.filename, "size": 2, "url": "https://cdn.discordapp.com/attachments/1/42/batch.json"}}},
			})
			assert.NoError(t, err)
			var data discordgo.ModalSubmitInteractionData
			assert.NoError(t, json.Unmarshal(raw, &data))
			if test.mutate != nil {
				test.mutate(&data)
			}
			result, err := ParseProblemBatchModalSource(data)
			if test.wantError != "" {
				assert.ErrorContains(t, err, test.wantError)
				assert.Empty(t, result)
				return
			}
			assert.NoError(t, err)
			if len(test.files) == 1 {
				if assert.NotNil(t, result.Attachment) {
					assert.Equal(t, "42", result.Attachment.ID)
				}
				assert.Empty(t, result.Text)
			} else {
				assert.Equal(t, test.text, result.Text)
				assert.Nil(t, result.Attachment)
			}
		})
	}
}

func TestJSONParseReceiptFitsDiscordLimit(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, message string
		count         int
	}{
		{name: "many entries", message: "unknown field", count: 100},
		{name: "large Unicode error", message: strings.Repeat("🚀", 2000), count: 1},
		{name: "many large errors", message: strings.Repeat("🚀", 2000), count: 100},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			issues := make([]JSONInputIssue, test.count)
			for index := range issues {
				issues[index] = JSONInputIssue{Entry: index + 1, Message: test.message}
			}
			result := FormatJSONParseResult(0, issues)
			assert.Contains(t, result, "Entry 1")
			assert.LessOrEqual(t, len(utf16.Encode([]rune(result))), 2000)
			if test.count > 1 {
				assert.Contains(t, result, "more issue(s)")
			}
		})
	}
}

func TestJSONModalSourceRejectsAnotherCommandRoute(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name  string
		route string
		parse func(discordgo.ModalSubmitInteractionData) (JSONModalSource, error)
	}{
		{name: "problem rejects AC route", route: ACBatchModalCustomID, parse: ParseProblemBatchModalSource},
		{name: "AC rejects problem route", route: ProblemBatchModalCustomID, parse: ParseACBatchModalSource},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := test.parse(discordgo.ModalSubmitInteractionData{CustomID: test.route})
			assert.ErrorContains(t, err, "form ID")
		})
	}
}
