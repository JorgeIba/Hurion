package mentor

import (
	"errors"
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

func mapBatchEntryErrors(issues []service_model.BatchEntryError) []batchInputIssue {
	mapped := make([]batchInputIssue, len(issues))
	for index, issue := range issues {
		mapped[index] = batchInputIssue{
			entry: issue.Entry,
			err:   issue.Err,
		}
	}
	return mapped
}

// mapCreateProblemParamsToCreateProblemModal prefills the creation form from supplied values.
func mapCreateProblemParamsToCreateProblemModal(params service_model.CreateProblemParams) interactiontransport.CreateProblemModal {
	return interactiontransport.CreateProblemModal{
		ClassCode:          params.ClassCode,
		ProblemModalFields: mapProblemValuesToModalFields(params.Name, params.URL, params.Difficulty, params.Topic, params.Priority),
	}
}

// mapUpdateProblemParamsToEditProblemModal prefills the editor from supplied values.
func mapUpdateProblemParamsToEditProblemModal(params service_model.UpdateProblemParams) interactiontransport.EditProblemModal {
	return interactiontransport.EditProblemModal{
		ProblemID:          params.ProblemID,
		ClassCode:          params.ClassCode,
		ProblemModalFields: mapProblemValuesToModalFields(params.Name, params.URL, params.Difficulty, params.Topic, params.Priority),
	}
}

// mapCreateProblemModalToCreateProblemParams maps parsed form values into a create request.
// Values remain unchanged for service normalization and validation.
func mapCreateProblemModalToCreateProblemParams(modal interactiontransport.CreateProblemModal) service_model.CreateProblemParams {
	return service_model.CreateProblemParams{
		Name:       modal.Name,
		URL:        modal.URL,
		ClassCode:  modal.ClassCode,
		Difficulty: modal.Difficulty,
		Topic:      modal.Topic,
		Priority:   modal.Priority,
	}
}

// mapEditProblemModalToUpdateProblemParams maps parsed form values into an update
// request, including the target and destination class code. Values still require
// service normalization and validation before writing.
func mapEditProblemModalToUpdateProblemParams(modal interactiontransport.EditProblemModal) service_model.UpdateProblemParams {
	return service_model.UpdateProblemParams{
		ProblemID:  modal.ProblemID,
		Name:       modal.Name,
		URL:        modal.URL,
		ClassCode:  modal.ClassCode,
		Difficulty: modal.Difficulty,
		Topic:      modal.Topic,
		Priority:   modal.Priority,
	}
}

func mapProblemValuesToModalFields(name, url string, difficulty, topic, priority *string) interactiontransport.ProblemModalFields {
	return interactiontransport.ProblemModalFields{
		Name: name, URL: url, Difficulty: difficulty, Topic: topic, Priority: priority,
	}
}

func mapBatchParseIssues(issues []batchInputIssue) []discord.JSONInputIssue {
	locations := make([]discord.JSONInputIssue, len(issues))

	for index, issue := range issues {
		locations[index] = discord.JSONInputIssue{
			Entry:   issue.entry,
			Line:    issue.line,
			Column:  issue.column,
			Message: issue.err.Error(),
		}
	}

	return locations
}

func mapJSONReadIssue(err error) discord.JSONInputIssue {
	if syntax, ok := errors.AsType[*jsonDocumentError](err); ok {
		return discord.JSONInputIssue{Line: syntax.line, Column: syntax.column, Message: syntax.err.Error()}
	}
	return discord.JSONInputIssue{Message: err.Error()}
}
