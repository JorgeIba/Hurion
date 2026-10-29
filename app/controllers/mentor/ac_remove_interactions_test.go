package mentor

import (
	"errors"
	"testing"

	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/providers/mock_providers"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRemoveACInteraction(t *testing.T) {
	for _, tc := range []struct {
		name, target                                        string
		mentor, admin, missing, dbFailure, invalid, noGuild bool
		permissionErr                                       error
		denied                                              bool
	}{
		{name: "self"},
		{name: "explicit self", target: "123"},
		{name: "mentor", target: "456", mentor: true},
		{name: "admin", target: "456", admin: true},
		{name: "unauthorized", target: "456", denied: true},
		{name: "permission failure", target: "456", permissionErr: errors.New("offline"), denied: true},
		{name: "missing AC", missing: true},
		{name: "database failure", dbFailure: true},
		{name: "invalid problem", invalid: true},
		{name: "outside server", noGuild: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mock := gomock.NewController(t)
			store := mock_data.NewMockDataStore(mock)
			validator := mock_providers.NewMockDiscordValidator(mock)
			client := &batchInteractionClient{}
			ctrl := NewACInteractionController(services.NewServices(store), validator, nil, dispatcher.NewDiscordInteractionResponder(client))
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
				validator.EXPECT().IsMentor("guild", "123").Return(tc.mentor, tc.permissionErr)
				if !tc.mentor && tc.permissionErr == nil {
					validator.EXPECT().IsAdmin("guild", "123").Return(tc.admin, nil)
				}
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
			ctrl.HandleRemoveACInteraction(event)
			require.Len(t, client.responses, 1)
			require.Equal(t, discordgo.MessageFlagsEphemeral, client.responses[0].Data.Flags)
			if tc.invalid || tc.noGuild {
				require.Empty(t, client.edits)
				return
			}
			require.Len(t, client.edits, 1)
			result := *client.edits[0].Content
			if tc.denied || tc.missing || tc.dbFailure {
				require.NotContains(t, result, "Removed AC")
			} else {
				require.Contains(t, result, "Removed AC for problem `5` from <@"+target+">.")
			}
		})
	}
}
