package mentor

import (
	"encoding/json/jsontext"

	"github.com/JorgeIba/Hurion/app/controllers/adapters"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

type acInput struct {
	UserID    string `json:"user_id"`
	ProblemID int64  `json:"problem_id"`
}

func parseACBatchInput(document jsontext.Value) (batchInput[acInput], []batchInputIssue) {
	return parseBatchInput[acInput](document)
}

func validateACInput(input acInput) error {
	_, err := service_model.NormalizeAndValidateCreateACParams(mapACInputToCreateACParams(input))
	if err != nil {
		return serviceerr.NewValidationError("invalid AC values", err)
	}
	return nil
}

func mapACInputToCreateACParams(input acInput) service_model.CreateACParams {
	return service_model.CreateACParams{
		UserID:    input.UserID,
		ProblemID: input.ProblemID,
	}
}

func mapACBatchInputToCreateACParams(entries batchInput[acInput]) []service_model.CreateACParams {
	params := make([]service_model.CreateACParams, len(entries))
	for index, entry := range entries {
		params[index] = mapACInputToCreateACParams(entry)
	}
	return params
}

func mapACBatchValidationIssues(entries batchInput[acInput], issues []batchInputIssue) []discord.JSONInputIssue {
	locations := make([]discord.JSONInputIssue, len(issues))
	for index, issue := range issues {
		var input acInput
		if issue.entry > 0 {
			input = entries[issue.entry-1]
		}
		locations[index] = discord.JSONInputIssue{
			Entry:   issue.entry,
			Message: adapters.MapError(issue.err, adapters.ActionMentorAddAC, input.UserID, input.ProblemID),
		}
	}
	return locations
}
