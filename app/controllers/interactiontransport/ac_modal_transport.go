package interactiontransport

import "github.com/bwmarrin/discordgo"

func BuildACBatchModalResponse() *discordgo.InteractionResponse {
	return buildJSONModalResponse(ACBatchModalID(), "AC batch", `[{"user_id":"123456789012345678","problem_id":42}]`)
}

func ParseACBatchModalSource(data discordgo.ModalSubmitInteractionData) (JSONModalSource, error) {
	if err := ValidateACBatchModalID(data.CustomID); err != nil {
		return JSONModalSource{}, err
	}
	return parseJSONModalSource(data)
}
