package mentor

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"io"
	"net/http"
	"testing"

	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/providers/mock_providers"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestACProposalEvidenceTravelsFromSubmissionThroughApproval(t *testing.T) {
	var screenshot bytes.Buffer
	require.NoError(t, png.Encode(&screenshot, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	for _, test := range []struct {
		name         string
		invalidImage bool
		downloadErr  error
	}{
		{name: "valid screenshot"},
		{name: "unsupported image", invalidImage: true},
		{name: "download fails", downloadErr: errors.New("connection lost")},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			db, err := data.NewDB(&data.DBConfig{DBPath: ":memory:", DBType: data.Test})
			require.NoError(t, err)
			t.Cleanup(db.CloseDB)
			classID, err := db.CreateClass(data.Class{ClassCode: "ac", Name: "AC class"})
			require.NoError(t, err)
			problemID, err := db.AddProblem(data.Problem{
				ClassID: classID,
				Name:    "Two Sum",
				URL:     "https://example.com/problem",
			})
			require.NoError(t, err)
			require.NoError(t, db.AddClassAccessRole(data.ClassAccessRole{ClassCode: "ac", RoleID: "student"}))
			svcs := services.NewServices(db)
			_, err = svcs.Solution.AddPendingACByPrincipal("original", service_model.Principal{
				User:  service_model.User{UserID: "student", DisplayName: "Student"},
				Roles: []string{"student"},
			}, problemID)
			require.NoError(t, err)
			body := screenshot.Bytes()
			if test.invalidImage {
				body = []byte("not an image")
			}
			downloads := 0
			client := &proposalInteractionClient{}
			controller := NewACInteractionController(svcs, nil, nil, dispatcher.NewDiscordInteractionResponder(client))
			controller.attachmentDownloader = interactiontransport.NewAttachmentDownloader(&http.Client{Transport: batchFileTransport(func(request *http.Request) (*http.Response, error) {
				downloads++
				require.Len(t, client.responses, 1, "acknowledge before downloading")
				if test.downloadErr != nil {
					return nil, test.downloadErr
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(body)),
					Request:    request,
				}, nil
			})})
			event := proposalCommand("with-evidence", problemID)
			command := event.ApplicationCommandData()
			command.Options[0].Options = append(command.Options[0].Options, &discordgo.ApplicationCommandInteractionDataOption{
				Name:  discordcommands.ACScreenshotOptionName,
				Type:  discordgo.ApplicationCommandOptionAttachment,
				Value: "proof",
			})
			command.Resolved = &discordgo.ApplicationCommandInteractionDataResolved{
				Attachments: map[string]*discordgo.MessageAttachment{"proof": {
					ID:       "proof",
					Filename: "evidence.png",
					URL:      "https://cdn.discordapp.com/attachments/1/2/evidence.png?ex=123&hm=signed",
					Size:     len(body),
				}},
			}
			event.Data = command
			controller.HandleProposeACInteraction(event)
			require.Equal(t, 1, downloads)
			require.Equal(t, discordgo.MessageFlagsEphemeral, client.responses[0].Data.Flags)
			if test.invalidImage || test.downloadErr != nil {
				require.Empty(t, client.followups, "invalid evidence must not publish a proposal card")
				require.Empty(t, client.deletedResponses)
				require.Len(t, client.edits, 1)
				require.NotEmpty(t, *client.edits[0].Content)
				userID, pendingProblemID, err := db.GetPendingConfirmation("original")
				require.NoError(t, err, "evidence preparation failure must retain the earlier proposal")
				require.Equal(t, "student", userID)
				require.Equal(t, problemID, pendingProblemID)
				_, _, err = db.GetPendingConfirmation("with-evidence")
				require.True(t, dataerr.IsNotFound(err))
				return
			}
			require.Empty(t, client.edits)
			require.Len(t, client.deletedResponses, 1)
			require.Len(t, client.followups, 1)
			card := client.followups[0]
			require.Len(t, card.Files, 1)
			require.Len(t, card.Attachments, 1)
			require.Len(t, card.Embeds, 1)
			require.Nil(t, card.Embeds[0].Image)
			upload, err := io.ReadAll(card.Files[0].Reader)
			require.NoError(t, err)
			require.Equal(t, screenshot.Bytes(), upload)
			_, _, err = db.GetPendingConfirmation("original")
			require.True(t, dataerr.IsNotFound(err))
			_, _, err = db.GetPendingConfirmation("with-evidence")
			require.NoError(t, err)
			validator := mock_providers.NewMockDiscordValidator(gomock.NewController(t))
			validator.EXPECT().IsMentor("guild", "mentor").Return(true, nil)
			controller.validator = validator
			approveID := card.Components[0].(discordgo.ActionsRow).Components[0].(discordgo.Button).CustomID
			controller.HandleApproveACProposalInteraction(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				ID:      "approve-evidence",
				Type:    discordgo.InteractionMessageComponent,
				GuildID: "guild",
				Member:  &discordgo.Member{User: &discordgo.User{ID: "mentor", Username: "Mentor"}},
				Message: &discordgo.Message{
					ID:          "public-card",
					Embeds:      card.Embeds,
					Attachments: []*discordgo.MessageAttachment{{ID: "stored-proof", Filename: card.Files[0].Name}},
				},
				Data: discordgo.MessageComponentInteractionData{
					CustomID:      approveID,
					ComponentType: discordgo.ButtonComponent,
				},
			}})
			require.Len(t, client.followups, 1, "resolution must not publish another card")
			require.Equal(t, 1, downloads, "resolution must not download evidence again")
			require.Len(t, client.edits, 1)
			edit := client.edits[0]
			require.Contains(t, (*edit.Embeds)[0].Description, "🎈 Approved by")
			require.Nil(t, (*edit.Embeds)[0].Image)
			require.Empty(t, *edit.Components)
			require.Len(t, *edit.Attachments, 1)
			require.Equal(t, "stored-proof", (*edit.Attachments)[0].ID)
			solved, err := db.HasUserSolvedProblem("student", problemID)
			require.NoError(t, err)
			require.True(t, solved)
		})
	}
}
