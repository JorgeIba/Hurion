package discord

import "github.com/bwmarrin/discordgo"

const ACBatchModalCustomID = "huron:ac:batch"

func BuildACBatchModalResponse() *discordgo.InteractionResponse {
	return buildJSONModalResponse(ACBatchModalCustomID, "AC batch", `[{"user_id":"123456789012345678","problem_id":42}]`)
}

func ParseACBatchModalSource(data discordgo.ModalSubmitInteractionData) (JSONModalSource, error) {
	return parseJSONModalSource(data, ACBatchModalCustomID)
}
