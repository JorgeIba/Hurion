package mentor

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
	"github.com/JorgeIba/Hurion/app/controllers/enrolled"
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/providers/mock_providers"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/mock_services"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type proposalInteractionClient struct {
	batchInteractionClient
	publicationErr error
	deletionErr    error
	editErr        error
	deliveries     []string
}

func (client *proposalInteractionClient) FollowupMessageCreate(interaction *discordgo.Interaction, wait bool, response *discordgo.WebhookParams, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	client.deliveries = append(client.deliveries, "publish")
	_, _ = client.batchInteractionClient.FollowupMessageCreate(interaction, wait, response, options...)
	return nil, client.publicationErr
}

func (client *proposalInteractionClient) InteractionResponseDelete(interaction *discordgo.Interaction, options ...discordgo.RequestOption) error {
	client.deliveries = append(client.deliveries, "delete acknowledgement")
	_ = client.batchInteractionClient.InteractionResponseDelete(interaction, options...)
	return client.deletionErr
}

func (client *proposalInteractionClient) InteractionResponseEdit(interaction *discordgo.Interaction, response *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	_, _ = client.batchInteractionClient.InteractionResponseEdit(interaction, response, options...)
	return nil, client.editErr
}

func proposalCommand(referenceID string, problemID int64) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:      referenceID,
		Type:    discordgo.InteractionApplicationCommand,
		GuildID: "guild",
		Member:  &discordgo.Member{User: &discordgo.User{ID: "student", Username: "Student"}, Roles: []string{"student"}},
		Data: discordgo.ApplicationCommandInteractionData{
			Name: "ac",
			Options: []*discordgo.ApplicationCommandInteractionDataOption{{
				Name: discordcommands.ACSubmitCommandName,
				Type: discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandInteractionDataOption{{
					Name:  discordcommands.ACProblemIDOptionName,
					Type:  discordgo.ApplicationCommandOptionString,
					Value: strconv.FormatInt(problemID, 10),
				}},
			}},
		},
	}}
}

func TestInvalidStringProblemIDStopsBeforePersistenceOrPublication(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name  string
		value string
	}{
		{name: "name without selection", value: "Two Sum"},
		{name: "decimal", value: "1.5"},
		{name: "empty"},
		{name: "integer overflow", value: "9223372036854775808"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			client := &proposalInteractionClient{}
			controller := NewACInteractionController(services.NewServices(store, nil), nil, dispatcher.NewDiscordInteractionResponder(client))
			event := proposalCommand("invalid-submission", 5)
			event.ApplicationCommandData().Options[0].Options[0].Value = test.value
			// No datastore expectations: invalid input must stop before any read/write.
			controller.HandleProposeACInteraction(event)
			require.Len(t, client.responses, 1)
			require.Equal(t, discordgo.InteractionResponseChannelMessageWithSource, client.responses[0].Type)
			require.Equal(t, discordgo.MessageFlagsEphemeral, client.responses[0].Data.Flags)
			require.Equal(t, "Enter a valid positive problem ID.", client.responses[0].Data.Content)
			require.Empty(t, client.followups)
			require.Empty(t, client.edits)
			require.Empty(t, client.deliveries)
		})
	}
}

func TestPrefixAndSlashProposalsShareRenewalIncludingPublicationFailures(t *testing.T) {
	for _, test := range []struct {
		name           string
		publicationErr error
		deletionErr    error
		editErr        error
	}{
		{name: "published without screenshot"},
		{name: "private acknowledgement deletion fails", deletionErr: errors.New("delete failed")},
		{name: "definitive Discord rejection", publicationErr: &discordgo.RESTError{Response: &http.Response{StatusCode: 400, Status: "400 Bad Request"}}},
		{name: "uncertain transport failure", publicationErr: errors.New("connection lost")},
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
			svcs := services.NewServices(db, nil)
			// Both transports must obtain the problem through SolutionService.
			svcs.Problem = nil
			fetcher := mock_providers.NewMockDiscordFetcher(gomock.NewController(t))
			fetcher.EXPECT().GetCachedMember("guild", "student").Return(&discordgo.Member{User: &discordgo.User{ID: "student"}, Roles: []string{"student"}}, nil).Times(2)
			out := make(chan dispatcher.MessageResponse, 2)
			prefix := makeCommandContext([]string{"!ac", fmt.Sprint(problemID)}, "student", "guild", "prefix-original", svcs, fetcher, out)
			enrolled.HandleProposeAC(prefix)
			require.Equal(t, dispatcher.Reaction, awaitMessage(t, out).Type)

			client := &proposalInteractionClient{
				publicationErr: test.publicationErr,
				deletionErr:    test.deletionErr,
			}
			controller := NewACInteractionController(svcs, nil, dispatcher.NewDiscordInteractionResponder(client))
			controller.HandleProposeACInteraction(proposalCommand("slash-reference", problemID))
			_, _, err = db.GetPendingConfirmation("prefix-original")
			require.True(t, dataerr.IsNotFound(err))
			userID, pendingProblemID, err := db.GetPendingConfirmation("slash-reference")
			require.NoError(t, err, "publication failure must not undo renewal")
			require.Equal(t, "student", userID)
			require.Equal(t, problemID, pendingProblemID)
			require.Len(t, client.responses, 1)
			require.Equal(t, discordgo.MessageFlagsEphemeral, client.responses[0].Data.Flags)
			require.Len(t, client.followups, 1)
			card := client.followups[0]
			require.Zero(t, card.Flags&discordgo.MessageFlagsEphemeral, "mentor evidence must be public")
			require.Len(t, card.Embeds, 1)
			require.Contains(t, card.Embeds[0].Description, "Two Sum")
			require.Contains(t, card.Embeds[0].Description, "<@student>")
			require.Empty(t, card.Files, "screenshots are optional")
			require.Empty(t, card.Attachments)
			if test.publicationErr != nil {
				require.Empty(t, client.deletedResponses)
				require.Equal(t, []string{"publish"}, client.deliveries)
				require.Len(t, client.edits, 1)
				require.Contains(t, *client.edits[0].Content, "has been replaced")
				require.Contains(t, *client.edits[0].Content, "submit the command again")
			} else {
				require.Equal(t, []string{"publish", "delete acknowledgement"}, client.deliveries)
				if test.deletionErr != nil {
					require.Len(t, client.edits, 1)
					require.Contains(t, *client.edits[0].Content, "published for mentor review")
				} else {
					require.Empty(t, client.edits)
				}
				require.Len(t, client.deletedResponses, 1)
				require.Equal(t, "slash-reference", client.deletedResponses[0].ID)
			}

			prefix.Message.ID = "prefix-retry"
			enrolled.HandleProposeAC(prefix)
			require.Equal(t, dispatcher.Reaction, awaitMessage(t, out).Type)
			_, _, err = db.GetPendingConfirmation("slash-reference")
			require.True(t, dataerr.IsNotFound(err))
			_, _, err = db.GetPendingConfirmation("prefix-retry")
			require.NoError(t, err)
		})
	}
}

func TestACProposalButtonsRespectAuthorityAndCurrentReference(t *testing.T) {
	for _, approve := range []bool{false, true} {
		for _, scenario := range []string{"mentor", "admin", "denied", "permission lookup failed", "replaced", "resolved", "public update failed", "persistence failed"} {
			action := "reject"
			if approve {
				action = "approve"
			}
			t.Run(action+"/"+scenario, func(t *testing.T) {
				t.Parallel()
				dbPath := ":memory:"
				if scenario == "persistence failed" {
					dbPath = filepath.Join(t.TempDir(), "proposal.sqlite")
				}
				db, err := data.NewDB(&data.DBConfig{DBPath: dbPath, DBType: data.Test})
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
				require.NoError(t, db.EnsureUser(data.User{UserID: "student"}))
				pending := data.PendingConfirmation{
					MessageID: "proposal-reference",
					UserID:    "student",
					ProblemID: problemID,
				}
				written, err := db.AddPendingConfirmation(pending)
				require.NoError(t, err)
				require.True(t, written)
				if scenario == "replaced" {
					pending.MessageID = "replacement-reference"
					written, err = db.AddPendingConfirmation(pending)
					require.NoError(t, err)
					require.True(t, written)
				} else if scenario == "resolved" {
					_, err = db.AcceptPendingConfirmation(pending.MessageID)
					require.NoError(t, err)
				}
				var failureDB *sql.DB
				if scenario == "persistence failed" {
					failureDB, err = sql.Open("sqlite3", dbPath)
					require.NoError(t, err)
					t.Cleanup(func() { _ = failureDB.Close() })
					query := `CREATE TRIGGER fail_resolution BEFORE DELETE ON pending_confirmations
                        BEGIN SELECT RAISE(ABORT, 'simulated consumption failure'); END;`
					if approve {
						query = `CREATE TRIGGER fail_resolution BEFORE INSERT ON solved_problems
                            BEGIN SELECT RAISE(ABORT, 'simulated AC write failure'); END;`
					}
					_, err = failureDB.Exec(query)
					require.NoError(t, err)
				}
				validator := mock_services.NewMockAuthorizer(gomock.NewController(t))
				var permissionErr error
				if scenario == "permission lookup failed" {
					permissionErr = errors.New("role lookup unavailable")
				}
				mentor := scenario != "admin" && scenario != "denied" && permissionErr == nil
				checks := 1
				if scenario == "resolved" || scenario == "replaced" {
					checks = 0
				}
				if scenario == "persistence failed" {
					checks = 2
				}
				validator.EXPECT().CanManageClasses(services.UserActor("actor"), gomock.Any()).Return(mentor || scenario == "admin", permissionErr).Times(checks)
				client := &proposalInteractionClient{}
				if scenario == "public update failed" {
					client.editErr = errors.New("Discord could not update the public card")
				}
				controller := NewACInteractionController(services.NewServices(db, validator), nil, dispatcher.NewDiscordInteractionResponder(client))
				customID := interactiontransport.ACProposalRejectID("proposal-reference")
				if approve {
					customID = interactiontransport.ACProposalApproveID("proposal-reference")
				}
				event := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
					ID:      "decision-interaction",
					Type:    discordgo.InteractionMessageComponent,
					GuildID: "guild",
					Member:  &discordgo.Member{User: &discordgo.User{ID: "actor", Username: "Mentor"}},
					Message: &discordgo.Message{ID: "public-card-message"},
					Data: discordgo.MessageComponentInteractionData{
						CustomID:      customID,
						ComponentType: discordgo.ButtonComponent,
					},
				}}
				if approve {
					controller.HandleApproveACProposalInteraction(event)
				} else {
					controller.HandleRejectACProposalInteraction(event)
				}
				success := scenario == "mentor" || scenario == "admin" || scenario == "public update failed"
				solved, err := db.HasUserSolvedProblem("student", problemID)
				require.NoError(t, err)
				require.Equal(t, scenario == "resolved" || (success && approve), solved)
				userID, actualProblemID, pendingErr := db.GetPendingConfirmation(pending.MessageID)
				if success || scenario == "resolved" {
					require.True(t, dataerr.IsNotFound(pendingErr))
				} else {
					require.NoError(t, pendingErr)
					require.Equal(t, "student", userID)
					require.Equal(t, problemID, actualProblemID)
				}
				if success {
					if scenario == "public update failed" {
						require.Len(t, client.followups, 1)
						require.Equal(t, discordgo.MessageFlagsEphemeral, client.followups[0].Flags)
						decision := "rejected"
						if approve {
							decision = "approved"
						}
						require.Contains(t, client.followups[0].Content, "was "+decision)
						require.Contains(t, client.followups[0].Content, "public card could not be updated")
					} else {
						require.Empty(t, client.followups)
					}
					require.Len(t, client.edits, 1)
					status := "Rejected"
					if approve {
						status = "Approved"
					}
					require.Contains(t, *client.edits[0].Content, status+" by")
					require.NotNil(t, client.edits[0].Components)
					require.Empty(t, *client.edits[0].Components)
				} else {
					require.Empty(t, client.edits, "failed decisions must not render public success")
					require.Len(t, client.followups, 1)
					require.Equal(t, discordgo.MessageFlagsEphemeral, client.followups[0].Flags)
					if scenario == "denied" {
						assert.Contains(t, client.followups[0].Content, "mentor or admin role is required")
					}
					if scenario == "replaced" || scenario == "resolved" {
						require.Contains(t, client.followups[0].Content, "resolved or replaced")
					}
				}
				if scenario == "persistence failed" {
					require.Contains(t, client.followups[0].Content, "Could not complete the decision")
					_, err = failureDB.Exec("DROP TRIGGER fail_resolution")
					require.NoError(t, err)
					event.ID = "retry-decision"
					if approve {
						controller.HandleApproveACProposalInteraction(event)
					} else {
						controller.HandleRejectACProposalInteraction(event)
					}
					require.Len(t, client.edits, 1, "the retry should resolve the retained proposal")
					require.Len(t, client.followups, 1)
					_, _, err = db.GetPendingConfirmation(pending.MessageID)
					require.True(t, dataerr.IsNotFound(err))
					solved, err = db.HasUserSolvedProblem("student", problemID)
					require.NoError(t, err)
					require.Equal(t, approve, solved)
				}

			})
		}
	}
}
