package mentor

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/JorgeIba/Hurion/app/authorization/authtest"
	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
	"github.com/JorgeIba/Hurion/app/presentation/discord"

	"github.com/JorgeIba/Hurion/app/dispatcher"
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
				name                                                                                                                               string
				otherActor, otherGuild, otherKind, missingMember, malformed, expired, expireDuringCheck, denied, permissionFails, ackFails, repeat bool
				wantPending                                                                                                                        bool
				wantEdits, wantFollowups                                                                                                           int
				wantContent                                                                                                                        string
			}{
				{name: "owner cancels", wantEdits: 1, wantContent: "Batch cancelled"},
				{name: "other actor", otherActor: true, wantPending: true, wantContent: discord.BatchUnavailableMessage},
				{name: "other guild", otherGuild: true, wantPending: true, wantContent: discord.BatchUnavailableMessage},
				{name: "other operation", otherKind: true, wantPending: true, wantContent: discord.BatchUnavailableMessage},
				{name: "missing member", missingMember: true, wantPending: true, wantContent: "identified server member"},
				{name: "malformed button", malformed: true, wantPending: true, wantContent: "invalid batch cancellation button"},
				{name: "expired", expired: true, wantContent: discord.BatchUnavailableMessage},
				{name: "expires during role check", expireDuringCheck: true},
				{name: "permissions revoked", denied: true, wantPending: true, wantFollowups: 1, wantContent: "mentor or admin"},
				{name: "permission lookup failed", permissionFails: true, wantPending: true, wantFollowups: 1, wantContent: "verifying permissions"},
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
					validator := authtest.NewMockRoleChecker(mockCtrl)
					if !test.otherActor && !test.otherGuild && !test.otherKind && !test.missingMember && !test.malformed && !test.expired && !test.ackFails {
						validator.EXPECT().IsMentor("guild", "mentor").DoAndReturn(func(string, string) (bool, error) {
							if test.expireDuringCheck {
								clock.now.Add(int64(5 * time.Minute))
							}
							if test.permissionFails {
								return false, errors.New("Discord unavailable")
							}
							return !test.denied, nil
						})
						if test.denied {
							validator.EXPECT().IsAdmin("guild", "mentor").Return(false, nil)
						}
					}
					client := &batchInteractionClient{}
					if test.ackFails {
						client.ackErr = errors.New("ack failed")
					}
					responder := dispatcher.NewDiscordInteractionResponder(client)
					var handle func(*discordgo.InteractionCreate)
					if family.problem {
						controller := NewProblemInteractionController(withTestInteractionAuthorization(nil, validator), responder)
						controller.validatedBatches = problemCache
						handle = controller.HandleCancelProblemBatchInteraction
					} else {
						controller := NewACInteractionController(withTestInteractionAuthorization(nil, validator), nil, responder)
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
							controller := NewACInteractionController(withTestInteractionAuthorization(nil, validator), nil, responder)
							controller.validatedBatches = acCache
							handle = controller.HandleCancelACBatchInteraction
						} else {
							route = string(discordcommands.CancelProblemBatch)
							controller := NewProblemInteractionController(withTestInteractionAuthorization(nil, validator), responder)
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
			validator := authtest.NewMockRoleChecker(gomock.NewController(t))
			// Both requests must pass the initial cache read before either can
			// claim the batch. This reproduces the race without timing sleeps.
			permissionChecks := make(chan struct{}, 2)
			releaseChecks := make(chan struct{})
			validator.EXPECT().IsMentor("guild", "mentor").DoAndReturn(func(string, string) (bool, error) {
				permissionChecks <- struct{}{}
				<-releaseChecks
				return true, nil
			}).Times(2)
			client := &batchInteractionClient{}
			responder := dispatcher.NewDiscordInteractionResponder(client)
			key := validatedBatchCacheKey{guildID: "guild", userID: "mentor", batchID: "batch"}
			var handle func(*discordgo.InteractionCreate)
			if family.problem {
				controller := NewProblemInteractionController(withTestInteractionAuthorization(nil, validator), responder)
				require.True(t, controller.validatedBatches.saveValidatedBatch(key, []service_model.PreparedProblem{{Entry: 1}}))
				handle = controller.HandleCancelProblemBatchInteraction
			} else {
				controller := NewACInteractionController(withTestInteractionAuthorization(nil, validator), nil, responder)
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
			<-permissionChecks
			<-permissionChecks
			close(releaseChecks)
			clicks.Wait()
			require.Len(t, client.responses, 2)
			for _, response := range client.responses {
				assert.Equal(t, discordgo.InteractionResponseDeferredMessageUpdate, response.Type)
			}
			require.Len(t, client.edits, 1, "the losing request must not overwrite the winner's receipt")
			assert.Equal(t, "Batch cancelled. No changes were applied.", *client.edits[0].Content)
			require.NotNil(t, client.edits[0].Components)
			assert.Empty(t, *client.edits[0].Components)
		})
	}
}

func TestDelayedCancellationAuthorizationFailurePreservesWinningReceipt(t *testing.T) {
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
			for _, lookupFails := range []bool{true, false} {
				name := "permissions revoked"
				if lookupFails {
					name = "permission lookup fails"
				}
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					validator := authtest.NewMockRoleChecker(gomock.NewController(t))
					slowCheckStarted := make(chan struct{})
					winningClickDone := make(chan struct{})
					// Start the slow request first and hold its permission result until the
					// second request has cancelled and delivered the original receipt.
					slowCheck := validator.EXPECT().IsMentor("guild", "mentor").DoAndReturn(func(string, string) (bool, error) {
						close(slowCheckStarted)
						<-winningClickDone
						if lookupFails {
							return false, errors.New("Discord lookup unavailable")
						}
						return false, nil
					})
					validator.EXPECT().IsMentor("guild", "mentor").Return(true, nil).After(slowCheck)
					if !lookupFails {
						validator.EXPECT().IsAdmin("guild", "mentor").Return(false, nil)
					}
					client := &batchInteractionClient{}
					responder := dispatcher.NewDiscordInteractionResponder(client)
					key := validatedBatchCacheKey{guildID: "guild", userID: "mentor", batchID: "batch"}
					var handle func(*discordgo.InteractionCreate)
					if family.problem {
						controller := NewProblemInteractionController(withTestInteractionAuthorization(nil, validator), responder)
						require.True(t, controller.validatedBatches.saveValidatedBatch(key, []service_model.PreparedProblem{{Entry: 1}}))
						handle = controller.HandleCancelProblemBatchInteraction
					} else {
						controller := NewACInteractionController(withTestInteractionAuthorization(nil, validator), nil, responder)
						require.True(t, controller.validatedBatches.saveValidatedBatch(key, []service_model.PreparedAC{{Entry: 1}}))
						handle = controller.HandleCancelACBatchInteraction
					}
					event := func(id string) *discordgo.InteractionCreate {
						return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
							ID: id, Type: discordgo.InteractionMessageComponent, GuildID: "guild",
							Member: &discordgo.Member{User: &discordgo.User{ID: "mentor"}},
							Data:   discordgo.MessageComponentInteractionData{CustomID: family.route + "|batch", ComponentType: discordgo.ButtonComponent},
						}}
					}
					slowClickDone := make(chan struct{})
					go func() { defer close(slowClickDone); handle(event("slow")) }()
					<-slowCheckStarted
					handle(event("winner"))
					close(winningClickDone)
					<-slowClickDone
					require.Len(t, client.responses, 2)
					for _, response := range client.responses {
						assert.Equal(t, discordgo.InteractionResponseDeferredMessageUpdate, response.Type)
					}
					require.Len(t, client.edits, 1, "authorization errors must not edit the shared preview")
					assert.Equal(t, "Batch cancelled. No changes were applied.", *client.edits[0].Content)
					require.NotNil(t, client.edits[0].Components)
					assert.Empty(t, *client.edits[0].Components)
					require.Len(t, client.followups, 1)
					assert.Equal(t, discordgo.MessageFlagsEphemeral, client.followups[0].Flags)
					assert.Empty(t, client.followups[0].AllowedMentions.Parse)
					expected := "mentor or admin"
					if lookupFails {
						expected = "verifying permissions"
					}
					assert.Contains(t, client.followups[0].Content, expected)
				})
			}
		})
	}
}
