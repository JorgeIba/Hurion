package discord

import (
	"encoding/json"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

func TestBatchPreviewCancelButtonRoundTripsAndFitsDiscordIDLimit(t *testing.T) {
	t.Parallel()
	for _, test := range []struct{ name, route string }{
		{name: "problem", route: ProblemBatchCancelCustomID},
		{name: "AC", route: ACBatchCancelCustomID},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			preview := BuildValidatedBatchPreview("Validated batch", test.route, "abcdefghijklmnopqrstuvwxyz")
			assert.Equal(t, "Validated batch", *preview.Content)
			assert.Empty(t, preview.AllowedMentions.Parse)
			if !assert.NotNil(t, preview.Components) || !assert.Len(t, *preview.Components, 1) {
				return
			}
			row := (*preview.Components)[0].(discordgo.ActionsRow)
			if !assert.Len(t, row.Components, 1) {
				return
			}
			button := row.Components[0].(discordgo.Button)
			assert.Equal(t, "Cancel", button.Label)
			assert.LessOrEqual(t, len(button.CustomID), 100)
			assert.Equal(t, test.route, ComponentRouteFromCustomID(button.CustomID))
			id, err := ParseBatchCancelID(discordgo.MessageComponentInteractionData{CustomID: button.CustomID, ComponentType: discordgo.ButtonComponent}, test.route)
			assert.NoError(t, err)
			assert.Equal(t, "abcdefghijklmnopqrstuvwxyz", id)
		})
	}
}

func TestBatchCancelIDRejectsMalformedOrWrongComponents(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, id string
		kind     discordgo.ComponentType
	}{
		{name: "missing ID", id: ProblemBatchCancelCustomID, kind: discordgo.ButtonComponent},
		{name: "empty ID", id: ProblemBatchCancelCustomID + "|", kind: discordgo.ButtonComponent},
		{name: "extra context", id: ProblemBatchCancelCustomID + "|id|extra", kind: discordgo.ButtonComponent},
		{name: "blank ID", id: ProblemBatchCancelCustomID + "| ", kind: discordgo.ButtonComponent},
		{name: "other route", id: ACBatchCancelCustomID + "|id", kind: discordgo.ButtonComponent},
		{name: "not a button", id: ProblemBatchCancelCustomID + "|id", kind: discordgo.SelectMenuComponent},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseBatchCancelID(discordgo.MessageComponentInteractionData{CustomID: test.id, ComponentType: test.kind}, ProblemBatchCancelCustomID)
			assert.Error(t, err)
		})
	}
}

func TestBatchCancellationExplicitlyClearsButtonsInDiscordPayload(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		cancelled bool
		want      string
	}{
		{name: "cancelled", cancelled: true, want: "Batch cancelled. No changes were applied."},
		{name: "unavailable", want: BatchUnavailableMessage},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			payload, err := json.Marshal(BuildBatchCancellationResult(test.cancelled))
			assert.NoError(t, err)
			var wire map[string]json.RawMessage
			assert.NoError(t, json.Unmarshal(payload, &wire))
			assert.JSONEq(t, "[]", string(wire["components"]))
			var content string
			assert.NoError(t, json.Unmarshal(wire["content"], &content))
			assert.Equal(t, test.want, content)
		})
	}
}
