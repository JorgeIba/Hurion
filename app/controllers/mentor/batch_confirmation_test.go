package mentor

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/providers"
	"github.com/JorgeIba/Hurion/app/providers/mock_providers"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/mock_services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestBatchConfirmationProtectsPendingOwnershipAndDeadline(t *testing.T) {
	t.Parallel()
	for _, family := range []struct {
		name    string
		problem bool
		edit    bool
		route   string
	}{
		{name: "problem", problem: true, route: string(discordcommands.ConfirmProblemBatch)},
		{name: "problem edit", problem: true, edit: true, route: string(discordcommands.ConfirmProblemEditBatch)},
		{name: "AC", route: string(discordcommands.ConfirmACBatch)},
	} {
		t.Run(family.name, func(t *testing.T) {
			t.Parallel()
			for _, test := range []struct {
				name                                                                                                                       string
				otherActor, otherGuild, otherKind, missingMember, malformed, expired, expireDuringCheck, denied, permissionFails, ackFails bool
				wantPending                                                                                                                bool
				wantEdits, wantFollowups                                                                                                   int
				wantContent                                                                                                                string
			}{
				{name: "other actor", otherActor: true, wantPending: true, wantContent: discord.BatchUnavailableMessage},
				{name: "other guild", otherGuild: true, wantPending: true, wantContent: discord.BatchUnavailableMessage},
				{name: "other operation", otherKind: true, wantPending: true, wantContent: discord.BatchUnavailableMessage},
				{name: "missing member", missingMember: true, wantPending: true, wantContent: "identified server member"},
				{name: "malformed button", malformed: true, wantPending: true, wantContent: "invalid batch action button"},
				{name: "expired", expired: true, wantContent: discord.BatchUnavailableMessage},
				{name: "expires during acknowledgement", expireDuringCheck: true},
				{name: "permissions revoked", denied: true, wantEdits: 1, wantContent: "mentor or admin"},
				{name: "permission lookup failed", permissionFails: true, wantEdits: 1, wantContent: "Could not"},
				{name: "failed acknowledgement", ackFails: true, wantPending: true},
			} {
				t.Run(test.name, func(t *testing.T) {
					t.Parallel()
					if family.edit && test.permissionFails {
						test.wantContent = "Submit the batch again"
					}
					clock := &draftClock{}
					clock.now.Store(time.Now().UnixNano())
					problemCache := newValidatedProblemBatchCache(clock)
					editCache := newValidatedProblemEditBatchCache(clock)
					acCache := newValidatedACBatchCache(clock)
					key := validatedBatchCacheKey{guildID: "guild", userID: "mentor", batchID: "batch"}
					var present func() bool
					if family.edit {
						assert.True(t, editCache.saveValidatedBatch(key, []service_model.PreparedProblemPatch{{Entry: 1, Params: service_model.PatchProblemParams{ProblemID: 1, PatchProblemFields: service_model.PatchProblemFields{Name: service_model.PatchField[string]{Set: true, Value: "New"}}}}}))
						present = func() bool { _, found := editCache.loadValidatedBatch(key); return found }
					} else if family.problem {
						assert.True(t, problemCache.saveValidatedBatch(key, []service_model.PreparedProblem{{Entry: 1, Params: service_model.CreateProblemParams{ClassCode: "basicos", Name: "New", URL: "https://example.com/new"}, Class: service_model.Class{ID: 1}}}))
						present = func() bool { _, found := problemCache.loadValidatedBatch(key); return found }
					} else {
						assert.True(t, acCache.saveValidatedBatch(key, []service_model.PreparedAC{{Entry: 1, User: service_model.User{UserID: "123"}, Problem: service_model.Problem{ID: 1}, Class: service_model.Class{ID: 1}}}))
						present = func() bool { _, found := acCache.loadValidatedBatch(key); return found }
					}
					if test.expired {
						clock.now.Add(int64(5 * time.Minute))
					}
					mockCtrl := gomock.NewController(t)
					validator := mock_services.NewMockAuthorizer(mockCtrl)
					fetcher := mock_providers.NewMockDiscordFetcher(mockCtrl)
					if !family.problem && (test.expireDuringCheck || test.denied || test.permissionFails) {
						fetcher.EXPECT().GetCachedMember("guild", "123").Return(&discordgo.Member{User: &discordgo.User{ID: "123"}, Roles: []string{"student"}}, nil)
					}
					if !test.otherActor && !test.otherGuild && !test.otherKind && !test.missingMember && !test.malformed && !test.expired && !test.ackFails && !test.expireDuringCheck {
						validator.EXPECT().CanManageClasses(services.UserActor("mentor"), gomock.Any()).DoAndReturn(func(service_model.Actor, ...string) (bool, error) {
							if test.permissionFails {
								return false, errors.New("Discord unavailable")
							}
							return !test.denied, nil
						})
					}
					client := &confirmationInteractionClient{batchInteractionClient: &batchInteractionClient{}}
					if test.expireDuringCheck {
						client.onRespond = func() { clock.now.Add(int64(5 * time.Minute)) }
					}
					if test.ackFails {
						client.ackErr = errors.New("ack failed")
					}
					responder := dispatcher.NewDiscordInteractionResponder(client)
					var handle func(*discordgo.InteractionCreate)
					if family.problem {
						controller := NewProblemInteractionController(confirmationTestServices(t, validator), responder)
						controller.validatedBatches = problemCache
						handle = controller.HandleConfirmProblemBatchInteraction
						if family.edit {
							controller.validatedEditBatches = editCache
							handle = controller.HandleConfirmProblemEditBatchInteraction
						}
					} else {
						controller := NewACInteractionController(confirmationTestServices(t, validator), fetcher, responder)
						controller.validatedBatches = acCache
						handle = controller.HandleConfirmACBatchInteraction
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
							route = string(discordcommands.ConfirmACBatch)
							controller := NewACInteractionController(confirmationTestServices(t, validator), fetcher, responder)
							controller.validatedBatches = acCache
							handle = controller.HandleConfirmACBatchInteraction
						} else {
							route = string(discordcommands.ConfirmProblemBatch)
							controller := NewProblemInteractionController(confirmationTestServices(t, validator), responder)
							controller.validatedBatches = problemCache
							handle = controller.HandleConfirmProblemBatchInteraction
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
				})
			}
		})
	}
}

func confirmationEvent(problem bool, action string) *discordgo.InteractionCreate {
	route := string(discordcommands.ConfirmACBatch)
	if problem {
		route = string(discordcommands.ConfirmProblemBatch)
	}
	if action == "Cancel" {
		route = string(discordcommands.CancelACBatch)
		if problem {
			route = string(discordcommands.CancelProblemBatch)
		}
	}
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionMessageComponent, GuildID: "guild", Member: &discordgo.Member{User: &discordgo.User{ID: "mentor"}},
		Data: discordgo.MessageComponentInteractionData{CustomID: route + "|batch", ComponentType: discordgo.ButtonComponent},
	}}
}

// Seed a real validated snapshot so race checks also exercise actual writes.
func setupConfirmationRace(t *testing.T, problem bool, validator services.Authorizer, fetcher providers.DiscordFetcher, client *confirmationInteractionClient) (confirm, cancel func(*discordgo.InteractionCreate), db *data.DB) {
	t.Helper()
	db, err := data.NewDB(&data.DBConfig{DBPath: ":memory:", DBType: data.Test})
	require.NoError(t, err)
	t.Cleanup(db.CloseDB)
	classID, err := db.CreateClass(data.Class{ClassCode: "basicos", Name: "Basicos"})
	require.NoError(t, err)
	svcs := services.NewServices(db, validator)
	responder := dispatcher.NewDiscordInteractionResponder(client)
	key := validatedBatchCacheKey{guildID: "guild", userID: "mentor", batchID: "batch"}
	if problem {
		prepared := []service_model.PreparedProblem{{Entry: 1, Params: service_model.CreateProblemParams{ClassCode: "basicos", Name: "New", URL: "https://example.com/new"}, Class: service_model.Class{ID: classID, ClassCode: "basicos"}}}
		controller := NewProblemInteractionController(svcs, responder)
		require.True(t, controller.validatedBatches.saveValidatedBatch(key, prepared))
		return controller.HandleConfirmProblemBatchInteraction, controller.HandleCancelProblemBatchInteraction, db
	}
	problemID, err := db.AddProblem(data.Problem{ClassID: classID, Name: "Existing", URL: "https://example.com/existing"})
	require.NoError(t, err)
	require.NoError(t, db.AddClassAccessRole(data.ClassAccessRole{ClassCode: "basicos", RoleID: "student"}))
	prepared := []service_model.PreparedAC{{Entry: 1, User: service_model.User{UserID: "123"}, Problem: service_model.Problem{ID: problemID}, Class: service_model.Class{ID: classID, ClassCode: "basicos"}}}
	controller := NewACInteractionController(svcs, fetcher, responder)
	require.True(t, controller.validatedBatches.saveValidatedBatch(key, prepared))
	return controller.HandleConfirmACBatchInteraction, controller.HandleCancelACBatchInteraction, db
}

func TestConcurrentBatchConfirmationHasOneWinner(t *testing.T) {
	t.Parallel()
	for _, problem := range []bool{true, false} {
		for _, otherAction := range []string{"Confirm", "Cancel"} {
			name := otherAction
			if problem {
				name = "problem/" + name
			} else {
				name = "AC/" + name
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				mockCtrl := gomock.NewController(t)
				validator := mock_services.NewMockAuthorizer(mockCtrl)
				fetcher := mock_providers.NewMockDiscordFetcher(mockCtrl)
				started := make(chan struct{}, 2)
				release := make(chan struct{})
				validator.EXPECT().CanManageClasses(services.UserActor("mentor"), gomock.Any()).Return(true, nil).MaxTimes(1)
				if !problem {
					count := 2
					if otherAction == "Cancel" {
						count = 1
					}
					fetcher.EXPECT().GetCachedMember("guild", "123").Return(&discordgo.Member{User: &discordgo.User{ID: "123"}, Roles: []string{"student"}}, nil).Times(count)
				}
				client := &confirmationInteractionClient{batchInteractionClient: &batchInteractionClient{}, onRespond: func() { started <- struct{}{}; <-release }}
				confirm, cancel, db := setupConfirmationRace(t, problem, validator, fetcher, client)
				var clicks sync.WaitGroup
				clicks.Go(func() { confirm(confirmationEvent(problem, "Confirm")) })
				clicks.Go(func() {
					if otherAction == "Confirm" {
						confirm(confirmationEvent(problem, "Confirm"))
					} else {
						cancel(confirmationEvent(problem, "Cancel"))
					}
				})
				<-started
				<-started
				close(release)
				clicks.Wait()
				require.Len(t, client.responses, 2)
				require.Len(t, client.edits, 1, "only the claimant may update the shared preview")
				receipt := *client.edits[0].Content
				cancelled := strings.Contains(receipt, "Batch cancelled")
				assert.Empty(t, *client.edits[0].Components)
				if problem {
					class, err := db.GetClassByClassCode("basicos")
					require.NoError(t, err)
					problems, err := db.GetProblemsByClass(class.ID)
					require.NoError(t, err)
					count := 1
					if cancelled {
						count = 0
					}
					assert.Len(t, problems, count)
				} else {
					solved, err := db.HasUserSolvedProblem("123", 1)
					require.NoError(t, err)
					assert.Equal(t, !cancelled, solved)
				}
			})
		}
	}
}

func TestDelayedConfirmationFailurePreservesWinner(t *testing.T) {
	t.Parallel()
	for _, problem := range []bool{true, false} {
		for _, failure := range []string{"authorization", "student resolution"} {
			if problem && failure == "student resolution" {
				continue
			}
			name := "AC/" + failure
			if problem {
				name = "problem/" + failure
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				mockCtrl := gomock.NewController(t)
				validator := mock_services.NewMockAuthorizer(mockCtrl)
				fetcher := mock_providers.NewMockDiscordFetcher(mockCtrl)
				started, release := make(chan struct{}), make(chan struct{})
				if failure == "authorization" {
					validator.EXPECT().CanManageClasses(services.UserActor("mentor"), gomock.Any()).DoAndReturn(func(service_model.Actor, ...string) (bool, error) {
						close(started)
						<-release
						return false, errors.New("Discord unavailable")
					})
					if !problem {
						fetcher.EXPECT().GetCachedMember("guild", "123").Return(&discordgo.Member{User: &discordgo.User{ID: "123"}, Roles: []string{"student"}}, nil)
					}
				} else {
					fetcher.EXPECT().GetCachedMember("guild", "123").DoAndReturn(func(string, string) (*discordgo.Member, error) {
						close(started)
						<-release
						return nil, errors.New("cache miss")
					})
					fetcher.EXPECT().FetchMember("guild", "123").Return(nil, errors.New("Discord unavailable"))
				}
				client := &confirmationInteractionClient{batchInteractionClient: &batchInteractionClient{}}
				confirm, cancel, _ := setupConfirmationRace(t, problem, validator, fetcher, client)
				done := make(chan struct{})
				go func() { defer close(done); confirm(confirmationEvent(problem, "Confirm")) }()
				<-started
				cancel(confirmationEvent(problem, "Cancel"))
				close(release)
				<-done
				require.Len(t, client.edits, 1)
				if failure == "authorization" {
					assert.Contains(t, *client.edits[0].Content, "Could not")
					require.Empty(t, client.followups)
				} else {
					assert.Equal(t, "Batch cancelled. No changes were applied.", *client.edits[0].Content)
					require.Len(t, client.followups, 1)
					assert.Equal(t, discordgo.MessageFlagsEphemeral, client.followups[0].Flags)
					assert.Empty(t, client.followups[0].AllowedMentions.Parse)
				}
			})
		}
	}
}

// Pause acknowledgement to exercise competing clicks before either can claim the batch.
type confirmationInteractionClient struct {
	*batchInteractionClient
	onRespond func()
}

func (c *confirmationInteractionClient) InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
	err := c.batchInteractionClient.InteractionRespond(i, r, options...)
	if c.onRespond != nil {
		c.onRespond()
	}
	return err
}
func confirmationTestServices(t *testing.T, auth services.Authorizer) *services.Services {
	t.Helper()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	class := data.Class{ID: 1, ClassCode: "basicos"}
	problem := data.Problem{ID: 1, ClassID: 1}
	store.EXPECT().GetClassesByClassCodes([]string{"basicos"}).Return([]data.Class{class}, nil).AnyTimes()
	store.EXPECT().GetProblemsByIDs([]int64{1}).Return([]data.Problem{problem}, nil).AnyTimes()
	store.EXPECT().GetClassByClassID(int64(1)).Return(class, nil).AnyTimes()
	store.EXPECT().GetACValidationData([]data.ACKey{{UserID: "123", ProblemID: 1}}).Return(map[data.ACKey]data.ACValidationData{{UserID: "123", ProblemID: 1}: {Problem: &problem, Class: &class, AccessRoleIDs: []string{"student"}}}, nil).AnyTimes()
	return services.NewServices(store, auth)
}
