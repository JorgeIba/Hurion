package mentor

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/services/mock_services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestBatchCancellationProtectsPendingOwnershipAndDeadline(t *testing.T) {
	t.Parallel()
	for _, family := range []struct {
		name    string
		problem bool
		route   string
	}{
		{name: "problem", problem: true, route: string(discordcommands.CancelProblemBatch)},
		{name: "AC", route: string(discordcommands.CancelACBatch)},
	} {
		t.Run(family.name, func(t *testing.T) {
			t.Parallel()
			for _, test := range []struct {
				name                                                                                                      string
				otherActor, otherGuild, otherKind, missingMember, malformed, expired, expireDuringCheck, ackFails, repeat bool
				wantPending                                                                                               bool
				wantEdits, wantFollowups                                                                                  int
				wantContent                                                                                               string
			}{
				{name: "owner cancels without consulting authorization", wantEdits: 1, wantContent: "Batch cancelled"},
				{name: "other actor", otherActor: true, wantPending: true, wantContent: discord.BatchUnavailableMessage},
				{name: "other guild", otherGuild: true, wantPending: true, wantContent: discord.BatchUnavailableMessage},
				{name: "other operation", otherKind: true, wantPending: true, wantContent: discord.BatchUnavailableMessage},
				{name: "missing member", missingMember: true, wantPending: true, wantContent: "identified server member"},
				{name: "malformed button", malformed: true, wantPending: true, wantContent: "invalid batch cancellation button"},
				{name: "expired", expired: true, wantContent: discord.BatchUnavailableMessage},

				{name: "failed acknowledgement", ackFails: true, wantPending: true},
				{name: "repeated cancellation", repeat: true, wantEdits: 1, wantContent: "Batch cancelled"},
			} {
				t.Run(test.name, func(t *testing.T) {
					t.Parallel()
					clock := &draftClock{}
					clock.now.Store(time.Now().UnixNano())
					problemCache := newValidatedProblemBatchCache(clock)
					acCache := newValidatedACBatchCache(clock)
					key := validatedBatchCacheKey{guildID: "guild", userID: "mentor", batchID: "batch"}
					var present func() bool
					if family.problem {
						assert.True(t, problemCache.saveValidatedBatch(key, []service_model.PreparedProblem{{Entry: 1}}))
						present = func() bool { _, found := problemCache.loadValidatedBatch(key); return found }
					} else {
						assert.True(t, acCache.saveValidatedBatch(key, []service_model.PreparedAC{{Entry: 1}}))
						present = func() bool { _, found := acCache.loadValidatedBatch(key); return found }
					}
					if test.expired {
						clock.now.Add(int64(5 * time.Minute))
					}
					mockCtrl := gomock.NewController(t)
					validator := mock_services.NewMockAuthorizer(mockCtrl)

					client := &batchInteractionClient{}
					if test.ackFails {
						client.ackErr = errors.New("ack failed")
					}
					responder := dispatcher.NewDiscordInteractionResponder(client)
					var handle func(*discordgo.InteractionCreate)
					if family.problem {
						controller := NewProblemInteractionController(permissionTestServices(t, validator), responder)
						controller.validatedBatches = problemCache
						handle = controller.HandleCancelProblemBatchInteraction
					} else {
						controller := NewACInteractionController(permissionTestServices(t, validator), nil, responder)
						controller.validatedBatches = acCache
						handle = controller.HandleCancelACBatchInteraction
					}
					actor, guild, route := "mentor", "guild", family.route
					if test.otherActor {
						actor = "other"
					}
					if test.otherGuild {
						guild = "other"
					}
					if test.otherKind {
						if family.problem {
							route = string(discordcommands.CancelACBatch)
							controller := NewACInteractionController(permissionTestServices(t, validator), nil, responder)
							controller.validatedBatches = acCache
							handle = controller.HandleCancelACBatchInteraction
						} else {
							route = string(discordcommands.CancelProblemBatch)
							controller := NewProblemInteractionController(permissionTestServices(t, validator), responder)
							controller.validatedBatches = problemCache
							handle = controller.HandleCancelProblemBatchInteraction
						}
					}
					customID := route + "|" + key.batchID
					if test.malformed {
						customID = route + "|"
					}
					member := &discordgo.Member{User: &discordgo.User{ID: actor}}
					if test.missingMember {
						member = nil
					}
					event := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{Type: discordgo.InteractionMessageComponent, GuildID: guild, Member: member, Data: discordgo.MessageComponentInteractionData{CustomID: customID, ComponentType: discordgo.ButtonComponent}}}
					handle(event)
					if assert.Len(t, client.edits, test.wantEdits) && test.wantEdits > 0 {
						assert.Contains(t, *client.edits[0].Content, test.wantContent)
						assert.Equal(t, discordgo.InteractionResponseDeferredMessageUpdate, client.responses[0].Type)
						if !test.wantPending {
							assert.NotNil(t, client.edits[0].Components)
							assert.Empty(t, *client.edits[0].Components)
						}
					} else if test.wantContent != "" && test.wantEdits == 0 && test.wantFollowups == 0 {
						if assert.Len(t, client.responses, 1) {
							assert.Contains(t, client.responses[0].Data.Content, test.wantContent)
							assert.Equal(t, discordgo.MessageFlagsEphemeral, client.responses[0].Data.Flags)
						}
					}
					if assert.Len(t, client.followups, test.wantFollowups) && test.wantFollowups > 0 {
						assert.Contains(t, client.followups[0].Content, test.wantContent)
						assert.Equal(t, discordgo.MessageFlagsEphemeral, client.followups[0].Flags)
						assert.Empty(t, client.followups[0].AllowedMentions.Parse)
						require.Len(t, client.responses, 1)
						assert.Equal(t, discordgo.InteractionResponseDeferredMessageUpdate, client.responses[0].Type)
					}
					assert.Equal(t, test.wantPending, present())
					if test.expireDuringCheck {
						require.Len(t, client.responses, 1)
						assert.Equal(t, discordgo.InteractionResponseDeferredMessageUpdate, client.responses[0].Type)
					}
					if test.repeat {
						handle(event)
						assert.Len(t, client.edits, 1)
						if assert.Len(t, client.responses, 2) {
							assert.Equal(t, discord.BatchUnavailableMessage, client.responses[1].Data.Content)
						}
					}
				})
			}
		})
	}
}

func TestConcurrentBatchCancellationOnlyWinnerUpdatesPreview(t *testing.T) {
	t.Parallel()
	for _, family := range []struct {
		name    string
		problem bool
		route   string
	}{
		{name: "problem", problem: true, route: string(discordcommands.CancelProblemBatch)},
		{name: "AC", route: string(discordcommands.CancelACBatch)},
	} {
		t.Run(family.name, func(t *testing.T) {
			t.Parallel()
			validator := mock_services.NewMockAuthorizer(gomock.NewController(t))
			client := &batchInteractionClient{}
			responder := dispatcher.NewDiscordInteractionResponder(client)
			key := validatedBatchCacheKey{guildID: "guild", userID: "mentor", batchID: "batch"}
			var handle func(*discordgo.InteractionCreate)
			if family.problem {
				controller := NewProblemInteractionController(permissionTestServices(t, validator), responder)
				require.True(t, controller.validatedBatches.saveValidatedBatch(key, []service_model.PreparedProblem{{Entry: 1}}))
				handle = controller.HandleCancelProblemBatchInteraction
			} else {
				controller := NewACInteractionController(permissionTestServices(t, validator), nil, responder)
				require.True(t, controller.validatedBatches.saveValidatedBatch(key, []service_model.PreparedAC{{Entry: 1}}))
				handle = controller.HandleCancelACBatchInteraction
			}
			var clicks sync.WaitGroup
			for _, interactionID := range []string{"first", "second"} {
				clicks.Go(func() {
					handle(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
						ID: interactionID, Type: discordgo.InteractionMessageComponent, GuildID: "guild",
						Member: &discordgo.Member{User: &discordgo.User{ID: "mentor"}},
						Data:   discordgo.MessageComponentInteractionData{CustomID: family.route + "|batch", ComponentType: discordgo.ButtonComponent},
					}})
				})
			}
			clicks.Wait()
			require.Len(t, client.responses, 2)
			for _, response := range client.responses {
				assert.Contains(t, []discordgo.InteractionResponseType{discordgo.InteractionResponseDeferredMessageUpdate, discordgo.InteractionResponseChannelMessageWithSource}, response.Type)
			}
			require.Len(t, client.edits, 1, "the losing request must not overwrite the winner's receipt")
			assert.Equal(t, "Batch cancelled. No changes were applied.", *client.edits[0].Content)
			require.NotNil(t, client.edits[0].Components)
			assert.Empty(t, *client.edits[0].Components)
		})
	}
}
