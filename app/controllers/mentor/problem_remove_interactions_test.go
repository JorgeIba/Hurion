package mentor

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/JorgeIba/Hurion/app/authorization"
	"github.com/JorgeIba/Hurion/app/authorization/mock_authorization"
	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func removalCommand(problemID int64) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID: "command", Type: discordgo.InteractionApplicationCommand, GuildID: "guild",
		Member: &discordgo.Member{User: &discordgo.User{ID: "mentor"}},
		Data: discordgo.ApplicationCommandInteractionData{
			Name: discordcommands.ProblemCommandName,
			Options: []*discordgo.ApplicationCommandInteractionDataOption{{
				Name: discordcommands.ProblemRemoveCommandName,
				Type: discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandInteractionDataOption{{
					Name:  discordcommands.ProblemIDOptionName,
					Type:  discordgo.ApplicationCommandOptionString,
					Value: strconv.FormatInt(problemID, 10),
				}},
			}},
		},
	}}
}

func removalClick(customID string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID: "click", Type: discordgo.InteractionMessageComponent, GuildID: "guild",
		Member: &discordgo.Member{User: &discordgo.User{ID: "mentor"}},
		Data:   discordgo.MessageComponentInteractionData{CustomID: customID, ComponentType: discordgo.ButtonComponent},
	}}
}

func TestProblemRemovalRequiresOwnedFreshConfirmation(t *testing.T) {
	for _, scenario := range []string{"confirm", "admin", "cancel", "other user", "other guild", "expired", "revoked", "lookup failure", "changed", "moved", "deleted", "failed acknowledgement", "failed preview delivery", "failed result delivery"} {
		t.Run(scenario, func(t *testing.T) {
			db, err := data.NewDB(&data.DBConfig{DBPath: ":memory:", DBType: data.Test})
			require.NoError(t, err)
			t.Cleanup(db.CloseDB)
			classID, err := db.CreateClass(data.Class{ClassCode: "class-a", Name: "Class A"})
			require.NoError(t, err)
			problemID, err := db.AddProblem(data.Problem{ClassID: classID, Name: "Two Sum", URL: "https://example.com/problem"})
			require.NoError(t, err)
			require.NoError(t, db.EnsureUser(data.User{UserID: "student", DisplayName: "Student"}))
			require.NoError(t, db.AddAC("student", problemID))
			source := mock_authorization.NewMockRoleSource(gomock.NewController(t))
			roles := []string{"mentor-a"}
			if scenario == "admin" {
				roles = []string{"admin"}
			}
			var lookupErr error
			source.EXPECT().FetchRoles("mentor").DoAndReturn(func(string) ([]string, error) { return roles, lookupErr }).AnyTimes()
			auth := authorization.New(source, "admin", "global", map[string]string{"class-a": "mentor-a"})
			client := &proposalInteractionClient{}
			ctrl := NewProblemInteractionController(services.NewServices(db, auth), dispatcher.NewDiscordInteractionResponder(client))
			clock := &draftClock{}
			clock.now.Store(time.Now().UnixNano())
			ctrl.pendingRemovals = newValidatedBatchCache[int64](clock)
			if scenario == "failed preview delivery" {
				client.editErr = errors.New("offline")
			}
			ctrl.HandleRemoveProblemInteraction(removalCommand(problemID))
			require.Len(t, client.responses, 1)
			require.Equal(t, discordgo.MessageFlagsEphemeral, client.responses[0].Data.Flags)
			require.Len(t, client.edits, 1)
			preview := client.edits[0]
			require.Contains(t, *preview.Content, "Two Sum")
			require.Contains(t, *preview.Content, "class-a")
			require.Contains(t, *preview.Content, "recorded ACs")
			_, err = db.GetProblemByID(problemID)
			require.NoError(t, err, "opening the preview must not delete")
			solved, err := db.HasUserSolvedProblem("student", problemID)
			require.NoError(t, err)
			require.True(t, solved)
			buttons := (*preview.Components)[0].(discordgo.ActionsRow).Components
			click := removalClick(buttons[0].(discordgo.Button).CustomID)
			switch scenario {
			case "cancel":
				click.Data = discordgo.MessageComponentInteractionData{CustomID: buttons[1].(discordgo.Button).CustomID, ComponentType: discordgo.ButtonComponent}
			case "other user":
				click.Member.User.ID = "intruder"
			case "other guild":
				click.GuildID = "other"
			case "expired":
				clock.now.Add(int64(5 * time.Minute))
			case "revoked":
				roles = nil
			case "lookup failure":
				lookupErr = errors.New("offline")
			case "moved":
				otherClassID, err := db.CreateClass(data.Class{ClassCode: "class-b", Name: "Class B"})
				require.NoError(t, err)
				require.NoError(t, db.ReplaceProblemByProblemID(problemID, data.Problem{ClassID: otherClassID, Name: "Two Sum", URL: "https://example.com/problem"}))
			case "failed result delivery":
				client.editErr = errors.New("offline")
			case "changed":
				require.NoError(t, db.ReplaceProblemByProblemID(problemID, data.Problem{ClassID: classID, Name: "Changed", URL: "https://example.com/problem"}))
			case "deleted":
				_, err = db.RemoveProblemByID(problemID)
				require.NoError(t, err)
			case "failed acknowledgement":
				client.ackErr = errors.New("offline")
			}
			if scenario == "cancel" {
				ctrl.HandleCancelProblemRemovalInteraction(click)
			} else {
				ctrl.HandleConfirmProblemRemovalInteraction(click)
			}
			_, err = db.GetProblemByID(problemID)
			if scenario == "confirm" || scenario == "admin" || scenario == "changed" || scenario == "failed result delivery" || scenario == "deleted" {
				require.True(t, dataerr.IsNotFound(err))
				solved, err = db.HasUserSolvedProblem("student", problemID)
				require.NoError(t, err)
				require.False(t, solved)
			} else {
				require.NoError(t, err)
				solved, err = db.HasUserSolvedProblem("student", problemID)
				require.NoError(t, err)
				require.True(t, solved, "unsuccessful removal must preserve recorded ACs")
			}
			if scenario == "confirm" || scenario == "admin" || scenario == "failed result delivery" || scenario == "lookup failure" || scenario == "moved" || scenario == "cancel" || scenario == "revoked" || scenario == "changed" || scenario == "deleted" {
				require.Len(t, client.edits, 2)
				require.Empty(t, *client.edits[1].Components)
				if scenario == "confirm" || scenario == "admin" || scenario == "changed" || scenario == "failed result delivery" {
					require.Contains(t, *client.edits[1].Content, "Deleted problem")
				} else {
					require.NotContains(t, *client.edits[1].Content, "Deleted problem")
				}
				ctrl.HandleConfirmProblemRemovalInteraction(click)
				require.Len(t, client.edits, 2, "repeated clicks must not execute or edit again")
			}
			if scenario == "other user" || scenario == "other guild" {
				ctrl.HandleConfirmProblemRemovalInteraction(removalClick(buttons[0].(discordgo.Button).CustomID))
				_, err = db.GetProblemByID(problemID)
				require.True(t, dataerr.IsNotFound(err), "an unrelated click must not consume the owner's confirmation")
			}
		})
	}
}

func TestProblemRemovalRejectsUnauthorizedPreview(t *testing.T) {
	for _, scenario := range []string{"global mentor", "role lookup error"} {
		t.Run(scenario, func(t *testing.T) {
			db, err := data.NewDB(&data.DBConfig{DBPath: ":memory:", DBType: data.Test})
			require.NoError(t, err)
			t.Cleanup(db.CloseDB)
			classID, err := db.CreateClass(data.Class{ClassCode: "class-a", Name: "Class A"})
			require.NoError(t, err)
			problemID, err := db.AddProblem(data.Problem{ClassID: classID, Name: "Two Sum", URL: "https://example.com"})
			require.NoError(t, err)
			source := mock_authorization.NewMockRoleSource(gomock.NewController(t))
			var lookupErr error
			if scenario == "role lookup error" {
				lookupErr = errors.New("offline")
			}
			source.EXPECT().FetchRoles("mentor").Return([]string{"global"}, lookupErr)
			auth := authorization.New(source, "admin", "global", map[string]string{"class-a": "mentor-a"})
			client := &batchInteractionClient{}
			ctrl := NewProblemInteractionController(services.NewServices(db, auth), dispatcher.NewDiscordInteractionResponder(client))
			ctrl.HandleRemoveProblemInteraction(removalCommand(problemID))
			require.Len(t, client.edits, 1)
			require.Nil(t, client.edits[0].Components)
			_, err = db.GetProblemByID(problemID)
			require.NoError(t, err)
		})
	}
}

func TestParseProblemRemoveRejectsInvalidInput(t *testing.T) {
	for _, value := range []any{"", "Two Sum", "0", "-1", "1.5", "9223372036854775808", float64(5)} {
		command := removalCommand(5).ApplicationCommandData()
		command.Options[0].Options[0].Value = value
		_, err := interactiontransport.ParseProblemRemoveOptions(command)
		require.Error(t, err)
	}
}
