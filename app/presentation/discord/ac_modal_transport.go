package discord

import (
	"fmt"

	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
)

const ACBatchModalCustomID = "huron:ac:batch"

func BuildACBatchModalResponse() *discordgo.InteractionResponse {
	return buildJSONModalResponse(ACBatchModalCustomID, "AC batch", `[{"user_id":"123456789012345678","problem_id":42}]`)
}

func ParseACBatchModalSource(data discordgo.ModalSubmitInteractionData) (JSONModalSource, error) {
	return parseJSONModalSource(data, ACBatchModalCustomID)
}

func FormatACBatchValidationResult(prepared []service_model.PreparedAC) string {
	students := make(map[string]struct{})
	alreadyRecorded := 0
	for _, entry := range prepared {
		students[entry.User.UserID] = struct{}{}
		if entry.AlreadyRecorded {
			alreadyRecorded++
		}
	}

	return fmt.Sprintf("AC batch validated. Entries: %d. Students: %d. Already recorded entries: %d. No changes were applied.", len(prepared), len(students), alreadyRecorded)
}
