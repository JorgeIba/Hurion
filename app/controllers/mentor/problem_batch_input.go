package mentor

import (
	"encoding/json/jsontext"
	"errors"
	"strings"

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
	if err := validateBatchProblemTopic(input.Topic, true); err != nil {
		return serviceerr.NewValidationError("invalid problem values", err)
	}
	_, err := service_model.NormalizeAndValidateCreateProblemParams(mapProblemInputToCreateProblemParams(input))
	if err != nil {
		return serviceerr.NewValidationError("invalid problem values", err)
	}
	return nil
}

func validateBatchProblemTopic(topic *string, required bool) error {
	if topic == nil {
		if required {
			return errors.New("problem topic is required")
		}
		return nil
	}

	value := strings.TrimSpace(*topic)
	if value == "" {
		return errors.New("problem topic must be non-blank")
	}
	if strings.EqualFold(value, service_model.DefaultTopic) {
		return errors.New("problem topic must be specific, not General")
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
	service_model.PatchProblemFields
}

func parseProblemEditBatchInput(document jsontext.Value) (batchInput[problemEditInput], []batchInputIssue) {
	return parseBatchInput[problemEditInput](document)
}

func mapProblemEditInputToPatchProblemParams(input problemEditInput) service_model.PatchProblemParams {
	return service_model.PatchProblemParams{
		ProblemID:          input.ProblemID,
		PatchProblemFields: input.PatchProblemFields,
	}
}

func validateProblemEditInput(input problemEditInput) error {
	if input.Topic.Set && input.Topic.Value != nil {
		if err := validateBatchProblemTopic(input.Topic.Value, false); err != nil {
			return serviceerr.NewValidationError("invalid problem values", err)
		}
	}
	_, err := service_model.NormalizeAndValidatePatchProblemParams(mapProblemEditInputToPatchProblemParams(input))
	if err != nil {
		return serviceerr.NewValidationError("invalid problem values", err)
	}
	return nil
}

func mapProblemEditBatchInputToPatchProblemParams(entries batchInput[problemEditInput]) []service_model.PatchProblemParams {
	params := make([]service_model.PatchProblemParams, len(entries))
	for index, entry := range entries {
		params[index] = mapProblemEditInputToPatchProblemParams(entry)
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
			Message: adapters.MapError(issue.err, adapters.ActionMentorUpdateProblem, input.ProblemID, input.ClassCode.Value, input.Name.Value),
		}
	}
	return locations
}

func mapProblemEditBatchExecutionIssues(batch []service_model.PreparedProblemPatch, issues []service_model.BatchEntryError) []discord.JSONInputIssue {
	entries := make(batchInput[problemEditInput], len(batch))
	for index, entry := range batch {
		entries[index] = problemEditInput{ProblemID: entry.Params.ProblemID, PatchProblemFields: entry.Params.PatchProblemFields}
	}
	return mapProblemEditBatchValidationIssues(entries, mapBatchEntryErrors(issues))
}
