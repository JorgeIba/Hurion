package discord

import (
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/require"
)

func TestACProposalEvidenceAppearsOnceAndSurvivesResolution(t *testing.T) {
	for _, withImage := range []bool{false, true} {
		for _, accepted := range []bool{false, true} {
			t.Run(fmt.Sprintf("screenshot=%t/approved=%t", withImage, accepted), func(t *testing.T) {
				t.Parallel()
				var image []byte
				if withImage {
					image = []byte("screenshot evidence")
				}
				card := BuildACProposalWebhookParams(service_model.User{
					UserID:      "123456789012345678",
					DisplayName: "Student",
				}, service_model.Problem{
					ID:   5,
					Name: "Two Sum",
					URL:  "https://example.com/two-sum",
				}, image, "image/png", "proof.png", "approve-reference", "reject-reference")
				require.Len(t, card.Embeds, 1)
				require.Nil(t, card.Embeds[0].Image, "the uploaded attachment must not also be embedded")
				require.Equal(t, "AC?", card.Embeds[0].Title)
				require.Equal(t, 0xFEE75C, card.Embeds[0].Color)
				require.Equal(t, "<@123456789012345678> submitted an AC proposal for: **[5: Two Sum](https://example.com/two-sum)**", card.Embeds[0].Description)
				row := card.Components[0].(discordgo.ActionsRow)
				require.Equal(t, "approve-reference", row.Components[0].(discordgo.Button).CustomID)
				require.Equal(t, "reject-reference", row.Components[1].(discordgo.Button).CustomID)
				approve := row.Components[0].(discordgo.Button)
				reject := row.Components[1].(discordgo.Button)
				require.NotNil(t, approve.Emoji)
				require.NotNil(t, reject.Emoji)
				require.Equal(t, "✅", approve.Emoji.Name)
				require.Equal(t, "✖️", reject.Emoji.Name)
				message := &discordgo.Message{Embeds: card.Embeds}
				if withImage {
					require.Len(t, card.Files, 1)
					upload, err := io.ReadAll(card.Files[0].Reader)
					require.NoError(t, err)
					require.Equal(t, image, upload)
					require.Len(t, card.Attachments, 1)
					message.Attachments = []*discordgo.MessageAttachment{{ID: "stored-evidence", Filename: "proof.png"}}
				} else {
					require.Empty(t, card.Files)
					require.Empty(t, card.Attachments)
				}
				originalDescription := message.Embeds[0].Description
				edit := BuildACProposalResolutionEdit(message, "Mentor", accepted)
				require.NotNil(t, edit.Components)
				require.Empty(t, *edit.Components)
				require.NotNil(t, edit.Embeds)
				resolved := (*edit.Embeds)[0]
				require.Nil(t, resolved.Image)
				require.NotContains(t, resolved.Description, acProposalPendingText)
				require.Contains(t, resolved.Description, "[5: Two Sum](https://example.com/two-sum)")
				require.Contains(t, resolved.Description, "<@123456789012345678>")
				require.Contains(t, resolved.Description, "Mentor")
				status := "🗣️ Rejected"
				title := "AC Rejected"
				color := 0xED4245
				if accepted {
					status = "🎈 Approved"
					title = "AC Approved"
					color = 0x57F287
				}
				require.Equal(t, title, resolved.Title)
				require.Equal(t, color, resolved.Color)
				require.Equal(t, 0xFEE75C, message.Embeds[0].Color, "resolution must not mutate the original embed")
				require.Contains(t, resolved.Description, status+" by")
				require.Equal(t, originalDescription, message.Embeds[0].Description, "rendering must not mutate the source message")
				if withImage {
					require.NotNil(t, edit.Attachments)
					require.Len(t, *edit.Attachments, 1)
					require.Equal(t, "stored-evidence", (*edit.Attachments)[0].ID)
					require.Equal(t, "proof.png", (*edit.Attachments)[0].Filename)
				} else {
					require.Nil(t, edit.Attachments)
				}
			})
		}
	}
}

func TestACProposalIdentifiesSameNamedStudentsWithoutAllowingMentionPings(t *testing.T) {
	descriptions := make([]string, 0, 2)
	for _, id := range []string{"123456789012345678", "234567890123456789"} {
		card := BuildACProposalWebhookParams(service_model.User{
			UserID:      id,
			DisplayName: "Student",
		}, service_model.Problem{ID: 5, Name: "Two Sum"}, nil, "", "", "approve", "reject")
		require.Contains(t, card.Embeds[0].Description, "<@"+id+">")
		descriptions = append(descriptions, card.Embeds[0].Description)
		payload, err := json.Marshal(card)
		require.NoError(t, err)
		var message map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(payload, &message))
		require.JSONEq(t, `{"parse":[],"replied_user":false}`, string(message["allowed_mentions"]))
	}
	require.NotEqual(t, descriptions[0], descriptions[1], "different accounts must remain distinguishable despite identical display names")
}
