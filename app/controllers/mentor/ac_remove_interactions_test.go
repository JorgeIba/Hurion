package mentor

import (
	"errors"
	"testing"

	"github.com/JorgeIba/Hurion/app/authorization"
	"github.com/JorgeIba/Hurion/app/authorization/mock_authorization"
	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/providers/mock_providers"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestRemoveACInteraction(t *testing.T) {
	type assertFunc func(t *testing.T, client *proposalInteractionClient, event *discordgo.InteractionCreate, target string)

	assertAcknowledged := func(t *testing.T, client *proposalInteractionClient) bool {
		if !assert.Len(t, client.responses, 1) {
			return false
		}
		response := client.responses[0]
		if !assert.NotNil(t, response.Data) {
			return false
		}
		assert.Equal(t, discordgo.MessageFlagsEphemeral, response.Data.Flags)
		return true
	}
	assertNoResult := func(t *testing.T, client *proposalInteractionClient, _ *discordgo.InteractionCreate, _ string) {
		if !assertAcknowledged(t, client) {
			return
		}
		assert.Empty(t, client.edits)
		assert.Empty(t, client.publications)
	}
	assertEditedContent := func(check func(t *testing.T, content string)) assertFunc {
		return func(t *testing.T, client *proposalInteractionClient, _ *discordgo.InteractionCreate, _ string) {
			if !assertAcknowledged(t, client) || !assert.Len(t, client.edits, 1) {
				return
			}
			content := client.edits[0].Content
			if !assert.NotNil(t, content) {
				return
			}
			check(t, *content)
			assert.Empty(t, client.publications)
		}
	}
	assertEditEquals := func(expected string) assertFunc {
		return assertEditedContent(func(t *testing.T, content string) {
			assert.Equal(t, expected, content)
		})
	}
	assertEditOmits := func(forbidden string) assertFunc {
		return assertEditedContent(func(t *testing.T, content string) {
			assert.NotContains(t, content, forbidden)
		})
	}
	assertRemoval := func(reference string, publicationFailure, deletionFailure bool) assertFunc {
		return func(t *testing.T, client *proposalInteractionClient, event *discordgo.InteractionCreate, target string) {
			if !assertAcknowledged(t, client) || !assert.Len(t, client.publications, 1) {
				return
			}
			assert.Equal(t, event.ChannelID, client.channelID)
			card := client.publications[0]
			assert.Empty(t, card.Content)
			assert.Nil(t, card.Reference)
			if !assert.Len(t, card.Embeds, 1) {
				return
			}
			assert.Equal(t, "AC Removed", card.Embeds[0].Title)
			assert.Equal(t, 0xED4245, card.Embeds[0].Color)
			assert.Equal(t, "Removed AC for problem `"+reference+"` from <@"+target+">.\n\nRemoved by <@123>", card.Embeds[0].Description)
			assert.Zero(t, card.Flags&discordgo.MessageFlagsEphemeral)
			assert.Empty(t, card.AllowedMentions.Parse)

			if publicationFailure {
				assert.Empty(t, client.deletedResponses)
				if assert.Len(t, client.edits, 1) && assert.NotNil(t, client.edits[0].Content) {
					assert.Contains(t, *client.edits[0].Content, "Removed AC for problem `"+reference+"` from <@"+target+">.")
					assert.Contains(t, *client.edits[0].Content, messageACRemovalPublicationUncertain)
				}
				return
			}

			assert.Len(t, client.deletedResponses, 1)
			if deletionFailure {
				if assert.Len(t, client.edits, 1) && assert.NotNil(t, client.edits[0].Content) {
					assert.Contains(t, *client.edits[0].Content, "Removed AC for problem `"+reference+"` from <@"+target+">.")
					assert.NotContains(t, *client.edits[0].Content, messageACRemovalPublicationUncertain)
				}
			} else {
				assert.Empty(t, client.edits)
			}
		}
	}

	for _, tc := range []struct {
		name, target, cacheName, dbName      string
		missing, dbFailure, invalid, noGuild bool
		noName                               bool
		roles                                []string
		permissionErr                        error
		lookupFailure                        bool
		denied                               bool
		publicationFailure, deletionFailure  bool
		assertFunc                           assertFunc
	}{
		{name: "self", assertFunc: assertRemoval("5: Two Sum", false, false)},
		{name: "name lookup failure", lookupFailure: true, assertFunc: assertRemoval("5", false, false)},
		{name: "publication failure", publicationFailure: true, assertFunc: assertRemoval("5: Two Sum", true, false)},
		{name: "acknowledgement deletion failure", deletionFailure: true, assertFunc: assertRemoval("5: Two Sum", false, true)},
		{name: "explicit self", target: "123", assertFunc: assertRemoval("5: Two Sum", false, false)},
		{name: "mentor", target: "456", roles: []string{"class-mentor"}, assertFunc: assertRemoval("5: Two Sum", false, false)},
		{name: "missing AC uses cached target name", target: "456", cacheName: "Cached Target", missing: true, roles: []string{"class-mentor"}, assertFunc: assertEditEquals("User `Cached Target` has not solved problem `5` yet")},
		{name: "missing AC falls back to database name", target: "456", dbName: "Known Target", missing: true, roles: []string{"class-mentor"}, assertFunc: assertEditEquals("User `Known Target` has not solved problem `5` yet")},
		{name: "missing AC preserves mention when name is unknown", target: "456", missing: true, noName: true, roles: []string{"class-mentor"}, assertFunc: assertEditEquals("User `<@456>` has not solved problem `5` yet")},
		{name: "missing AC treats Unknown database name as unresolved", target: "456", dbName: "Unknown", missing: true, roles: []string{"class-mentor"}, assertFunc: assertEditEquals("User `<@456>` has not solved problem `5` yet")},
		{name: "admin", target: "456", roles: []string{"admin"}, assertFunc: assertRemoval("5: Two Sum", false, false)},
		{name: "unauthorized", target: "456", denied: true, assertFunc: assertEditOmits("Removed AC")},
		{name: "global mentor cannot bypass class override", target: "456", roles: []string{"mentor"}, denied: true, assertFunc: assertEditOmits("Removed AC")},
		{name: "mentor of another class", target: "456", roles: []string{"other-class-mentor"}, denied: true, assertFunc: assertEditOmits("Removed AC")},
		{name: "permission failure", target: "456", permissionErr: errors.New("offline"), denied: true, assertFunc: assertEditOmits("Removed AC")},
		{name: "missing AC", missing: true, assertFunc: assertEditEquals("User `Student` has not solved problem `5` yet")},
		{name: "database failure", dbFailure: true, assertFunc: assertEditOmits("Removed AC")},
		{name: "invalid problem", invalid: true, assertFunc: assertNoResult},
		{name: "outside server", noGuild: true, assertFunc: assertNoResult},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mock := gomock.NewController(t)
			store := mock_data.NewMockDataStore(mock)
			roles := mock_authorization.NewMockRoleSource(mock)
			fetcher := mock_providers.NewMockDiscordFetcher(mock)
			auth := authorization.New(roles, "admin", "mentor", map[string]string{"class-a": "class-mentor"})
			client := &proposalInteractionClient{}
			if tc.publicationFailure {
				client.publicationErr = errors.New("publication failed")
			}
			if tc.deletionFailure {
				client.deletionErr = errors.New("deletion failed")
			}
			ctrl := NewACInteractionController(services.NewServices(store, auth), fetcher, dispatcher.NewDiscordInteractionResponder(client))
			event := proposalCommand("remove", 5)
			event.Member.User.ID = "123"
			event.Member.User.Username = "Student"
			event.ApplicationCommandData().Options[0].Name = discordcommands.ACRemoveCommandName
			if tc.target != "" {
				event.ApplicationCommandData().Options[0].Options = append(event.ApplicationCommandData().Options[0].Options, &discordgo.ApplicationCommandInteractionDataOption{
					Name:  "user",
					Type:  discordgo.ApplicationCommandOptionUser,
					Value: tc.target,
				})
			}
			if tc.invalid {
				event.ApplicationCommandData().Options[0].Options[0].Value = "-1"
			}
			if tc.noGuild {
				event.GuildID = ""
			}
			if tc.target != "" && tc.target != "123" {
				store.EXPECT().GetProblemByID(int64(5)).Return(data.Problem{ID: 5, ClassID: 7}, nil)
				store.EXPECT().GetClassByClassID(int64(7)).Return(data.Class{ClassCode: "class-a"}, nil)
				roles.EXPECT().FetchRoles("123").Return(tc.roles, tc.permissionErr)
			}

			target := tc.target
			if target == "" {
				target = "123"
			}
			if !tc.denied && !tc.invalid && !tc.noGuild {
				store.EXPECT().HasUserSolvedProblem(target, int64(5)).Return(!tc.missing, nil)
				if !tc.missing {
					var err error
					if tc.dbFailure {
						err = errors.New("write failed")
					}
					store.EXPECT().RemoveAC(target, int64(5)).Return(err)
				}
			}
			if !tc.denied && !tc.invalid && !tc.noGuild && !tc.missing && !tc.dbFailure {
				var lookupErr error
				if tc.lookupFailure {
					lookupErr = errors.New("offline")
				}
				name := "Two Sum"
				if tc.lookupFailure {
					name = ""
				}
				store.EXPECT().GetProblemByID(int64(5)).Return(data.Problem{ID: 5, Name: name}, lookupErr)
			}
			if tc.missing && tc.target != "" && tc.target != "123" {
				var member *discordgo.Member
				var cacheErr error
				if tc.cacheName != "" {
					member = &discordgo.Member{User: &discordgo.User{ID: tc.target, Username: tc.cacheName}}
				} else {
					cacheErr = errors.New("cache miss")
				}
				fetcher.EXPECT().GetCachedMember("guild", tc.target).Return(member, cacheErr)
				if tc.dbName != "" {
					store.EXPECT().GetUserByUserID(tc.target).Return(data.User{UserID: tc.target, DisplayName: tc.dbName}, nil)
				} else if tc.noName {
					store.EXPECT().GetUserByUserID(tc.target).Return(data.User{}, errors.New("not found"))
				}
			}
			ctrl.HandleRemoveACInteraction(event)
			tc.assertFunc(t, client, event, target)
		})
	}
}
