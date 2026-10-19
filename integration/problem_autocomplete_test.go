package integration

import (
	"fmt"
	"strconv"
	"testing"

	appbot "github.com/JorgeIba/Hurion/app/bot"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type autocompleteResultRecorder struct {
	dispatcher.InteractionResponder
	responses       []*discordgo.InteractionResponse
	privateMessages []string
}

func (r *autocompleteResultRecorder) Respond(_ *discordgo.Interaction, response *discordgo.InteractionResponse) error {
	r.responses = append(r.responses, response)
	return nil
}

func (r *autocompleteResultRecorder) RespondPrivately(_ *discordgo.Interaction, message string) error {
	r.privateMessages = append(r.privateMessages, message)
	return nil
}

func TestIntegration_ProblemAutocompleteReadsChannelClassWithoutChangingACState(t *testing.T) {
	t.Parallel()
	for _, route := range []struct {
		command    string
		subcommand string
		option     string
	}{
		{command: "ac", subcommand: "submit", option: "id"},
		{command: "problem", subcommand: "edit", option: "problem"},
	} {
		for _, test := range []struct {
			name   string
			linked bool
		}{
			{name: "linked channel supplies catalog", linked: true},
			{name: "unlinked channel supplies no suggestions"},
		} {
			t.Run(route.command+"/"+test.name, func(t *testing.T) {
				t.Parallel()
				env := setupIntegrationEnv(t)
				classA := env.SeedClass(t, "basicos", "Basics")
				classB := env.SeedClass(t, "destino", "Destination")
				if test.linked {
					env.SeedClassroom(t, "channel", "basicos")
				}
				require.NoError(t, env.DB.AddClassAccessRole(data.ClassAccessRole{
					ClassCode: "basicos",
					RoleID:    "required-student-role",
				}))
				pendingID := env.SeedProblem(t, "Pending Sum", "https://example.com/pending", classA, "easy", "Arrays")
				recordedID := env.SeedProblem(t, "Recorded Sum", "https://example.com/recorded", classA, "easy", "Arrays")
				openID := env.SeedProblem(t, "Open Sum", "https://example.com/open", classA, "easy", "Arrays")
				env.SeedProblem(t, "Other Sum", "https://example.com/other", classB, "easy", "Arrays")
				env.SeedUser(t, "123", "Original student")
				// Existing history can remain after a student's access role is revoked.
				env.SeedPendingConfirmation(t, "original-pending", "123", pendingID)
				env.SeedAC(t, "123", recordedID)
				before, err := env.DB.GetProblemsByClass(classA)
				require.NoError(t, err)

				options := []*discordgo.ApplicationCommandInteractionDataOption{nil}
				if route.command == "problem" {
					// Destination is deliberately different from the channel's class.
					options = append(options, &discordgo.ApplicationCommandInteractionDataOption{
						Name:  "class",
						Type:  discordgo.ApplicationCommandOptionString,
						Value: "destino",
					})
				}
				options = append(options, &discordgo.ApplicationCommandInteractionDataOption{
					Name:    route.option,
					Type:    discordgo.ApplicationCommandOptionString,
					Value:   "sum",
					Focused: true,
				})
				event := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
					ID:        "autocomplete",
					Type:      discordgo.InteractionApplicationCommandAutocomplete,
					GuildID:   "guild",
					ChannelID: "channel",
					Member:    &discordgo.Member{User: &discordgo.User{ID: "123", Username: "New name"}},
					Data: discordgo.ApplicationCommandInteractionData{
						Name: route.command,
						Options: []*discordgo.ApplicationCommandInteractionDataOption{{
							Name:    route.subcommand,
							Type:    discordgo.ApplicationCommandOptionSubCommand,
							Options: options,
						}},
					},
				}}
				responder := &autocompleteResultRecorder{}
				bot := appbot.NewBot(env.Services, nil, env.Dispatcher, responder, appbot.BotConfig{GuildID: "guild"})
				// There is no mentor/AC command controller: this is a read interaction.
				bot.HandleInteractionEvent(nil, event, nil, nil)

				require.Len(t, responder.responses, 1)
				response := responder.responses[0]
				assert.Equal(t, discordgo.InteractionApplicationCommandAutocompleteResult, response.Type)
				require.NotNil(t, response.Data)
				want := []*discordgo.ApplicationCommandOptionChoice{}
				if test.linked {
					want = []*discordgo.ApplicationCommandOptionChoice{
						{Name: fmt.Sprintf("%d: Pending Sum", pendingID), Value: strconv.FormatInt(pendingID, 10)},
						{Name: fmt.Sprintf("%d: Recorded Sum", recordedID), Value: strconv.FormatInt(recordedID, 10)},
						{Name: fmt.Sprintf("%d: Open Sum", openID), Value: strconv.FormatInt(openID, 10)},
					}
				}
				assert.Equal(t, want, response.Data.Choices)
				assert.Empty(t, response.Data.Content)
				assert.Empty(t, responder.privateMessages)
				assert.Empty(t, env.Dispatcher.GetSentMessages())

				after, err := env.DB.GetProblemsByClass(classA)
				require.NoError(t, err)
				assert.Equal(t, before, after)
				user, err := env.DB.GetUserByUserID("123")
				require.NoError(t, err)
				assert.Equal(t, "Original student", user.DisplayName)
				pendingUser, pendingProblem, err := env.DB.GetPendingConfirmation("original-pending")
				require.NoError(t, err)
				assert.Equal(t, "123", pendingUser)
				assert.Equal(t, pendingID, pendingProblem)
				newPending, err := env.DB.ExistsPendingConfirmation(event.ID)
				require.NoError(t, err)
				assert.False(t, newPending)
				for _, problemID := range []int64{pendingID, recordedID, openID} {
					solved, err := env.DB.HasUserSolvedProblem("123", problemID)
					require.NoError(t, err)
					assert.Equal(t, problemID == recordedID, solved)
				}
			})
		}
	}
}
