package mentor

import (
	"errors"
	"fmt"

	controllers_common "github.com/JorgeIba/Hurion/app/controllers/common"
	"github.com/JorgeIba/Hurion/app/providers"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// batchPreparer bridges parsed inputs and live Discord credentials to read-only
// service preparation. Returned errors belong to the controller; indexed issues
// come from batch validation. Command registration and confirmation are separate steps.
type batchPreparer struct {
	batch     *services.BatchService
	validator providers.DiscordValidator
	fetcher   providers.DiscordFetcher
}

func (preparer *batchPreparer) prepareProblemSubmission(guildID, actorID string, entries []problemBatchEntry) ([]service_model.PreparedProblem, []service_model.BatchEntryError, error) {
	if err := preparer.authorizeProblemBatch(guildID, actorID); err != nil {
		return nil, nil, err
	}

	inputs := make([]service_model.CreateProblemParams, len(entries))
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
	}

	prepared, issues := preparer.batch.PrepareProblemBatch(inputs)
	return prepared, issues, nil
}

func (preparer *batchPreparer) prepareACSubmission(guildID, actorID string, entries []acBatchEntry) ([]service_model.PreparedAC, []service_model.BatchEntryError, error) {
	if err := preparer.authorizeACBatch(guildID, actorID); err != nil {
		return nil, nil, err
	}

	inputs := make([]service_model.CreateACParams, len(entries))
	for index, entry := range entries {
		inputs[index] = service_model.CreateACParams{
			UserID:    entry.input.UserID,
			ProblemID: entry.input.ProblemID,
		}
	}

	studentsByID := make(map[string]service_model.Principal)
	for index, input := range inputs {
		userID := service_model.NormalizeACUserID(input.UserID)
		// Missing references are passed to batch for required-field validation.
		if userID == "" {
			continue
		}

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
