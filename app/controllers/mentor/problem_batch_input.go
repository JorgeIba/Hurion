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

func mapProblemBatchExecutionIssues(batch []service_model.PreparedProblem, issues []service_model.BatchEntryError) []discord.JSONInputIssue {
	entries := make(batchInput[problemInput], len(batch))
	for index, entry := range batch {
		entries[index] = problemInput{Class: entry.Params.ClassCode, Name: entry.Params.Name, URL: entry.Params.URL}
	}
	return mapProblemBatchValidationIssues(entries, mapBatchEntryErrors(issues))
}

type problemEditInput struct {
	ProblemID int64 `json:"problem_id"`
	problemInput
}

func parseProblemEditBatchInput(document jsontext.Value) (batchInput[problemEditInput], []batchInputIssue) {
	return parseBatchInput[problemEditInput](document)
}

func mapProblemEditInputToReplaceProblemParams(input problemEditInput) service_model.ReplaceProblemParams {
	return service_model.ReplaceProblemParams{
		ProblemID:  input.ProblemID,
		ClassCode:  input.Class,
		Name:       input.Name,
		URL:        input.URL,
		Difficulty: input.Difficulty,
		Topic:      input.Topic,
		Priority:   input.Priority,
	}
}

func validateProblemEditInput(input problemEditInput) error {
	_, err := service_model.NormalizeAndValidateReplaceProblemParams(mapProblemEditInputToReplaceProblemParams(input))
	if err != nil {
		return serviceerr.NewValidationError("invalid problem values", err)
	}
	return nil
}

func mapProblemEditBatchInputToReplaceProblemParams(entries batchInput[problemEditInput]) []service_model.ReplaceProblemParams {
	params := make([]service_model.ReplaceProblemParams, len(entries))
	for index, entry := range entries {
		params[index] = mapProblemEditInputToReplaceProblemParams(entry)
	}
	return params
}

func mapProblemEditBatchValidationIssues(entries batchInput[problemEditInput], issues []batchInputIssue) []discord.JSONInputIssue {
	locations := make([]discord.JSONInputIssue, len(issues))
	for index, issue := range issues {
		var input problemEditInput
		if issue.entry > 0 {
			input = entries[issue.entry-1]
		}
		locations[index] = discord.JSONInputIssue{
			Entry:   issue.entry,
			Message: adapters.MapError(issue.err, adapters.ActionMentorUpdateProblem, input.ProblemID, input.Class, input.Name),
		}
	}
	return locations
}

func mapProblemEditBatchExecutionIssues(batch []service_model.PreparedProblemReplacement, issues []service_model.BatchEntryError) []discord.JSONInputIssue {
	entries := make(batchInput[problemEditInput], len(batch))
	for index, entry := range batch {
		entries[index] = problemEditInput{ProblemID: entry.Params.ProblemID, problemInput: problemInput{Class: entry.Params.ClassCode, Name: entry.Params.Name}}
	}
	return mapProblemEditBatchValidationIssues(entries, mapBatchEntryErrors(issues))
}
