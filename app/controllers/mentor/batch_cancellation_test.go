package mentor

import (
	"errors"
	"testing"
	"time"

	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/providers/mock_providers"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestBatchCancellationProtectsPendingOwnershipAndDeadline(t *testing.T) {
	t.Parallel()
	for _, family := range []struct {
		name    string
		problem bool
		route   string
	}{
		{name: "problem", problem: true, route: discord.ProblemBatchCancelCustomID},
		{name: "AC", route: discord.ACBatchCancelCustomID},
	} {
		t.Run(family.name, func(t *testing.T) {
			t.Parallel()
			for _, test := range []struct {
				name                                                                                                                               string
				otherActor, otherGuild, otherKind, missingMember, malformed, expired, expireDuringCheck, denied, permissionFails, ackFails, repeat bool
				wantPending                                                                                                                        bool
				wantEdits                                                                                                                          int
				wantContent                                                                                                                        string
			}{
				{name: "owner cancels", wantEdits: 1, wantContent: "Batch cancelled"},
				{name: "other actor", otherActor: true, wantPending: true, wantContent: discord.BatchUnavailableMessage},
				{name: "other guild", otherGuild: true, wantPending: true, wantContent: discord.BatchUnavailableMessage},
				{name: "other operation", otherKind: true, wantPending: true, wantContent: discord.BatchUnavailableMessage},
				{name: "missing member", missingMember: true, wantPending: true, wantContent: "identified server member"},
				{name: "malformed button", malformed: true, wantPending: true, wantContent: "invalid batch cancellation button"},
				{name: "expired", expired: true, wantContent: discord.BatchUnavailableMessage},
				{name: "expires during role check", expireDuringCheck: true, wantEdits: 1, wantContent: discord.BatchUnavailableMessage},
				{name: "permissions revoked", denied: true, wantPending: true, wantEdits: 1, wantContent: "mentor or admin"},
				{name: "permission lookup failed", permissionFails: true, wantPending: true, wantEdits: 1, wantContent: "verifying permissions"},
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
					validator := mock_providers.NewMockDiscordValidator(mockCtrl)
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
						controller := NewProblemInteractionController(nil, validator, responder)
						controller.validatedBatches = problemCache
						handle = controller.HandleCancelProblemBatchInteraction
					} else {
						controller := NewACInteractionController(nil, validator, nil, responder)
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
							route = discord.ACBatchCancelCustomID
							controller := NewACInteractionController(nil, validator, nil, responder)
							controller.validatedBatches = acCache
							handle = controller.HandleCancelACBatchInteraction
						} else {
							route = discord.ProblemBatchCancelCustomID
							controller := NewProblemInteractionController(nil, validator, responder)
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
					} else if test.wantContent != "" && test.wantEdits == 0 {
						if assert.Len(t, client.responses, 1) {
							assert.Contains(t, client.responses[0].Data.Content, test.wantContent)
							assert.Equal(t, discordgo.MessageFlagsEphemeral, client.responses[0].Data.Flags)
						}
					}
					assert.Equal(t, test.wantPending, present())
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
