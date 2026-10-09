package mentor

import (
	"errors"
	"fmt"

	"github.com/JorgeIba/Hurion/app/controllers/adapters"
	controllers_common "github.com/JorgeIba/Hurion/app/controllers/common"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/providers"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// batchPreparer bridges parsed inputs and live Discord credentials to read-only
// service preparation. Returned errors belong to the controller; indexed issues
// describe invalid batch input. Command registration and confirmation are separate steps.
type batchPreparer struct {
	batch     *services.BatchService
	validator providers.DiscordValidator
	fetcher   providers.DiscordFetcher
}

func problemBatchValidationReceipt(entries []problemBatchEntry, issues []service_model.BatchEntryError) string {
	locations := make([]discord.JSONInputIssue, len(issues))
	for index, issue := range issues {
		var input problemBatchInput
		if issue.Entry > 0 {
			input = entries[issue.Entry-1].input
		}
		locations[index] = discord.JSONInputIssue{Entry: issue.Entry, Message: adapters.MapError(issue.Err, adapters.ActionMentorAddProblem, input.Class, input.URL, input.Name)}
	}
	return discord.FormatBatchValidationErrors(locations)
}

func acBatchValidationReceipt(entries []acBatchEntry, issues []service_model.BatchEntryError) string {
	locations := make([]discord.JSONInputIssue, len(issues))
	for index, issue := range issues {
		var input acBatchInput
		if issue.Entry > 0 {
			input = entries[issue.Entry-1].input
		}
		locations[index] = discord.JSONInputIssue{Entry: issue.Entry, Message: adapters.MapError(issue.Err, adapters.ActionMentorAddAC, input.UserID, input.ProblemID)}
	}
	return discord.FormatBatchValidationErrors(locations)
}

func (preparer *batchPreparer) prepareProblemSubmission(guildID, actorID string, entries []problemBatchEntry) ([]service_model.PreparedProblem, []service_model.BatchEntryError, error) {
	if err := preparer.authorizeProblemBatch(guildID, actorID); err != nil {
		return nil, nil, err
	}
	return preparer.prepareProblemEntries(entries)
}

// prepareProblemEntries requires the caller to have authorized the actor.
func (preparer *batchPreparer) prepareProblemEntries(entries []problemBatchEntry) ([]service_model.PreparedProblem, []service_model.BatchEntryError, error) {
	if err := services.ValidateBatchSize(len(entries)); err != nil {
		return nil, []service_model.BatchEntryError{{Err: err}}, nil
	}

	inputs := make([]service_model.CreateProblemParams, len(entries))
	var inputIssues []service_model.BatchEntryError
	for index, entry := range entries {
		value := entry.input
		inputs[index] = service_model.CreateProblemParams{
			ClassCode:  value.Class,
			Name:       value.Name,
			URL:        value.URL,
			Difficulty: value.Difficulty,
			Topic:      value.Topic,
			Priority:   value.Priority,
		}
		normalized, err := service_model.NormalizeAndValidateCreateProblemParams(inputs[index])
		if err != nil {
			inputIssues = append(inputIssues, service_model.BatchEntryError{Entry: index + 1, Err: serviceerr.NewValidationError("invalid problem values", err)})
		}
		inputs[index] = normalized
	}
	if len(inputIssues) > 0 {
		return nil, inputIssues, nil
	}

	prepared, issues := preparer.batch.PrepareProblemBatch(inputs)
	return prepared, issues, nil
}

func (preparer *batchPreparer) prepareACSubmission(guildID, actorID string, entries []acBatchEntry) ([]service_model.PreparedAC, []service_model.BatchEntryError, error) {
	if err := preparer.authorizeACBatch(guildID, actorID); err != nil {
		return nil, nil, err
	}
	return preparer.prepareACEntries(guildID, entries)
}

// prepareACEntries requires the caller to have authorized the actor.
func (preparer *batchPreparer) prepareACEntries(guildID string, entries []acBatchEntry) ([]service_model.PreparedAC, []service_model.BatchEntryError, error) {
	if err := services.ValidateBatchSize(len(entries)); err != nil {
		return nil, []service_model.BatchEntryError{{Err: err}}, nil
	}

	inputs := make([]service_model.CreateACParams, len(entries))
	var inputIssues []service_model.BatchEntryError
	for index, entry := range entries {
		inputs[index] = service_model.CreateACParams{
			UserID:    entry.input.UserID,
			ProblemID: entry.input.ProblemID,
		}
		normalized, err := service_model.NormalizeAndValidateCreateACParams(inputs[index])
		if err != nil {
			inputIssues = append(inputIssues, service_model.BatchEntryError{Entry: index + 1, Err: serviceerr.NewValidationError("invalid AC values", err)})
		}
		inputs[index] = normalized
	}
	if len(inputIssues) > 0 {
		return nil, inputIssues, nil
	}

	studentsByID := make(map[string]service_model.Principal)
	for index, input := range inputs {
		userID := input.UserID

		if _, resolved := studentsByID[userID]; resolved {
			continue
		}

		principal, err := controllers_common.GetMemberPrincipal(preparer.fetcher, guildID, userID)
		if err != nil {
			return nil, nil, fmt.Errorf("could not resolve student %s at entry %d: %w", userID, index+1, err)
		}
		studentsByID[userID] = *principal
	}

	prepared, issues := preparer.batch.PrepareACBatch(inputs, studentsByID)
	return prepared, issues, nil
}

func validateBatchActor(guildID, actorID string) error {
	if guildID == "" || actorID == "" {
		return errors.New("Batch commands require an identified server member.")
	}
	return nil
}

func (preparer *batchPreparer) authorizeProblemBatch(guildID, actorID string) error {
	if err := validateBatchActor(guildID, actorID); err != nil {
		return err
	}

	allowed, err := canManageProblems(preparer.validator, guildID, actorID)
	if err != nil {
		return errors.New(messageErrVerifying)
	}
	if !allowed {
		return errors.New(messageErrProblemPermissions)
	}
	return nil
}

func (preparer *batchPreparer) authorizeACBatch(guildID, actorID string) error {
	if err := validateBatchActor(guildID, actorID); err != nil {
		return err
	}

	allowed, err := canManageACs(preparer.validator, guildID, actorID)
	if err != nil {
		return errors.New(messageErrVerifying)
	}
	if !allowed {
		return errors.New(messageErrACPermissions)
	}
	return nil
}
