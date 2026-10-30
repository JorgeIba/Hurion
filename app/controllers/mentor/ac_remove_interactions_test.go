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
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestRemoveACInteraction(t *testing.T) {
	for _, tc := range []struct {
		name, target                         string
		missing, dbFailure, invalid, noGuild bool
		lookupFailure                        bool
		roles                                []string
		permissionErr                        error
		denied                               bool
	}{
		{name: "self"},
		{name: "name lookup failure", lookupFailure: true},
		{name: "explicit self", target: "123"},
		{name: "mentor", target: "456", roles: []string{"class-mentor"}},
		{name: "admin", target: "456", roles: []string{"admin"}},
		{name: "unauthorized", target: "456", denied: true},
		{name: "global mentor cannot bypass class override", target: "456", roles: []string{"mentor"}, denied: true},
		{name: "mentor of another class", target: "456", roles: []string{"other-class-mentor"}, denied: true},
		{name: "permission failure", target: "456", permissionErr: errors.New("offline"), denied: true},
		{name: "missing AC", missing: true},
		{name: "database failure", dbFailure: true},
		{name: "invalid problem", invalid: true},
		{name: "outside server", noGuild: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mock := gomock.NewController(t)
			store := mock_data.NewMockDataStore(mock)
			roles := mock_authorization.NewMockRoleSource(mock)
			auth := authorization.New(roles, "admin", "mentor", map[string]string{"class-a": "class-mentor"})
			client := &batchInteractionClient{}
			ctrl := NewACInteractionController(services.NewServices(store, auth), nil, dispatcher.NewDiscordInteractionResponder(client))
			event := proposalCommand("remove", 5)
			event.Member.User.ID = "123"
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
			ctrl.HandleRemoveACInteraction(event)
			if !assert.Len(t, client.responses, 1) {
				return
			}
			assert.Equal(t, discordgo.MessageFlagsEphemeral, client.responses[0].Data.Flags)
			if tc.invalid || tc.noGuild {
				assert.Empty(t, client.edits)
				return
			}
			if !assert.Len(t, client.edits, 1) {
				return
			}
			result := *client.edits[0].Content
			if tc.denied || tc.missing || tc.dbFailure {
				assert.NotContains(t, result, "Removed AC")
			} else {
				reference := "5: Two Sum"
				if tc.lookupFailure {
					reference = "5"
				}
				assert.Equal(t, "Removed AC for problem `"+reference+"` from <@"+target+">.", result)
			}
		})
	}
}
