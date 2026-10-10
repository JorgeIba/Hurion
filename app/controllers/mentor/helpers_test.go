package mentor

import (
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCancelControllersRejectNonButtonComponents(t *testing.T) {
	for _, problem := range []bool{false, true} {
		client := &batchInteractionClient{}
		responder := dispatcher.NewDiscordInteractionResponder(client)
		event := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionMessageComponent,
			Data: discordgo.MessageComponentInteractionData{CustomID: interactiontransport.ACBatchCancelID("id"), ComponentType: discordgo.SelectMenuComponent},
		}}
		if problem {
			ctrl := &ProblemInteractionController{responder: responder}
			ctrl.HandleCancelProblemBatchInteraction(event)
		} else {
			ctrl := &ACInteractionController{responder: responder}
			ctrl.HandleCancelACBatchInteraction(event)
		}
		if assert.Len(t, client.responses, 1) {
			assert.Equal(t, "invalid batch cancellation button", client.responses[0].Data.Content)
		}
	}
}
