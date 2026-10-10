package mentor

import (
	"encoding/json/jsontext"

	"github.com/JorgeIba/Hurion/app/controllers/adapters"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

type problemInput struct {
	Class      string  `json:"class"`
	Name       string  `json:"name"`
	URL        string  `json:"url"`
	Difficulty *string `json:"difficulty"`
	Topic      *string `json:"topic"`
	Priority   *string `json:"priority"`
}

func parseProblemBatchInput(document jsontext.Value) (batchInput[problemInput], []batchInputIssue) {
	return parseBatchInput[problemInput](document)
}

func validateProblemInput(input problemInput) error {
	_, err := service_model.NormalizeAndValidateCreateProblemParams(mapProblemInputToCreateProblemParams(input))
	if err != nil {
		return serviceerr.NewValidationError("invalid problem values", err)
	}
	return nil
}

func mapProblemInputToCreateProblemParams(input problemInput) service_model.CreateProblemParams {
	return service_model.CreateProblemParams{
		ClassCode:  input.Class,
		Name:       input.Name,
		URL:        input.URL,
		Difficulty: input.Difficulty,
		Topic:      input.Topic,
		Priority:   input.Priority,
	}
}

func mapProblemBatchInputToCreateProblemParams(entries batchInput[problemInput]) []service_model.CreateProblemParams {
	params := make([]service_model.CreateProblemParams, len(entries))
	for index, entry := range entries {
		params[index] = mapProblemInputToCreateProblemParams(entry)
	}
	return params
}

func mapProblemBatchValidationIssues(entries batchInput[problemInput], issues []batchInputIssue) []discord.JSONInputIssue {
	locations := make([]discord.JSONInputIssue, len(issues))
	for index, issue := range issues {
		var input problemInput
		if issue.entry > 0 {
			input = entries[issue.entry-1]
		}
		locations[index] = discord.JSONInputIssue{
			Entry:   issue.entry,
			Message: adapters.MapError(issue.err, adapters.ActionMentorAddProblem, input.Class, input.URL, input.Name),
		}
	}
	return locations
}
