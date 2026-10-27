package mentor

import (
	"encoding/json/jsontext"
	"strings"

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

func mapACBatchValidationIssues(entries batchInput[acInput], issues []batchInputIssue, students map[string]service_model.UserWithRoles) []discord.JSONInputIssue {
	locations := make([]discord.JSONInputIssue, len(issues))
	for index, issue := range issues {
		var input acInput
		if issue.entry > 0 {
			input = entries[issue.entry-1]
		}

		userLabel := input.UserID
		if student, ok := students[service_model.NormalizeACUserID(input.UserID)]; ok {
			if name := strings.TrimSpace(student.DisplayName); name != "" && name != "Unknown" {
				userLabel = strings.NewReplacer("`", "'", "\n", " ", "\r", " ").Replace(name)
			}
		}

		locations[index] = discord.JSONInputIssue{
			Entry:   issue.entry,
			Message: adapters.MapError(issue.err, adapters.ActionMentorAddAC, userLabel, input.ProblemID),
		}
	}
	return locations
}

func mapACBatchExecutionIssues(batch []service_model.PreparedAC, issues []service_model.BatchEntryError, students map[string]service_model.UserWithRoles) []discord.JSONInputIssue {
	entries := make(batchInput[acInput], len(batch))
	for index, entry := range batch {
		entries[index] = acInput{UserID: entry.User.UserID, ProblemID: entry.Problem.ID}
	}
	return mapACBatchValidationIssues(entries, mapBatchEntryErrors(issues), students)
}
