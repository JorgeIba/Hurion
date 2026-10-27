package mentor

import (
	"testing"

	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCancelControllersRejectNonButtonComponents(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		problem bool
	}{
		{name: "AC batch"},
		{name: "problem batch", problem: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client := &batchInteractionClient{}
			responder := dispatcher.NewDiscordInteractionResponder(client)
			event := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionMessageComponent,
				Data: discordgo.MessageComponentInteractionData{CustomID: interactiontransport.ACBatchCancelID("id"), ComponentType: discordgo.SelectMenuComponent},
			}}
			if tc.problem {
				ctrl := &ProblemInteractionController{responder: responder}
				ctrl.HandleCancelProblemBatchInteraction(event)
			} else {
				ctrl := &ACInteractionController{responder: responder}
				ctrl.HandleCancelACBatchInteraction(event)
			}
			if assert.Len(t, client.responses, 1) {
				assert.Equal(t, "invalid batch cancellation button", client.responses[0].Data.Content)
			}
		})
	}
}

// Cancellation is local to the actor's preview; neither stored resources nor
// permissions may be queried. Strict mocks reject either lookup.
func permissionTestServices(t *testing.T, validator services.Authorizer) *services.Services {
	t.Helper()
	store := mock_data.NewMockDataStore(gomock.NewController(t))
	return services.NewServices(store, validator)
}
