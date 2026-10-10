package discord_test

import (
	"encoding/json"
	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBatchPreviewCancelButtonRoundTripsAndFitsDiscordIDLimit(t *testing.T) {
	t.Parallel()
	for _, test := range []struct{ name, route, confirmRoute string }{
		{name: "problem", route: string(discordcommands.CancelProblemBatch), confirmRoute: string(discordcommands.ConfirmProblemBatch)},
		{name: "AC", route: string(discordcommands.CancelACBatch), confirmRoute: string(discordcommands.ConfirmACBatch)},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			preview := discord.BuildValidatedBatchPreview("Validated batch", test.confirmRoute+"|abcdefghijklmnopqrstuvwxyz", test.route+"|abcdefghijklmnopqrstuvwxyz")
			assert.Contains(t, *preview.Content, "Validated batch")
			assert.Empty(t, preview.AllowedMentions.Parse)
			if !assert.NotNil(t, preview.Components) || !assert.Len(t, *preview.Components, 1) {
				return
			}
			row := (*preview.Components)[0].(discordgo.ActionsRow)
			if !assert.Len(t, row.Components, 2) {
				return
			}
			confirm := row.Components[0].(discordgo.Button)
			assert.Equal(t, "Confirm", confirm.Label)
			assert.Equal(t, test.confirmRoute+"|abcdefghijklmnopqrstuvwxyz", confirm.CustomID)
			assert.LessOrEqual(t, len(confirm.CustomID), 100)
			button := row.Components[1].(discordgo.Button)
			assert.Equal(t, "Cancel", button.Label)
			assert.LessOrEqual(t, len(button.CustomID), 100)
			id, err := parseCancelIDForTest(button.CustomID, test.route)
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
		{name: "missing ID", id: string(discordcommands.CancelProblemBatch), kind: discordgo.ButtonComponent},
		{name: "empty ID", id: string(discordcommands.CancelProblemBatch) + "|", kind: discordgo.ButtonComponent},
		{name: "extra context", id: string(discordcommands.CancelProblemBatch) + "|id|extra", kind: discordgo.ButtonComponent},
		{name: "blank ID", id: string(discordcommands.CancelProblemBatch) + "| ", kind: discordgo.ButtonComponent},
		{name: "other route", id: string(discordcommands.CancelACBatch) + "|id", kind: discordgo.ButtonComponent},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := interactiontransport.ParseProblemBatchCancelID(test.id)
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
		{name: "unavailable", want: discord.BatchUnavailableMessage},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			payload, err := json.Marshal(discord.BuildBatchCancellationResult(test.cancelled))
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

func parseCancelIDForTest(id, route string) (string, error) {
	if route == string(discordcommands.CancelProblemBatch) {
		return interactiontransport.ParseProblemBatchCancelID(id)
	}
	return interactiontransport.ParseACBatchCancelID(id)
}
