package discord

import "github.com/JorgeIba/Hurion/app/services/service_model"

// MapCreateProblemParamsToCreateProblemModal prefills the creation form from supplied values.
func MapCreateProblemParamsToCreateProblemModal(params service_model.CreateProblemParams) CreateProblemModal {
	return CreateProblemModal{
		ClassCode:          params.ClassCode,
		ProblemModalFields: mapProblemValuesToModalFields(params.Name, params.URL, params.Difficulty, params.Topic, params.Priority),
	}
}

// MapUpdateProblemParamsToEditProblemModal prefills the editor from supplied values.
func MapUpdateProblemParamsToEditProblemModal(params service_model.UpdateProblemParams) EditProblemModal {
	return EditProblemModal{
		ProblemID:          params.ProblemID,
		ClassCode:          params.ClassCode,
		ProblemModalFields: mapProblemValuesToModalFields(params.Name, params.URL, params.Difficulty, params.Topic, params.Priority),
	}
}

// MapCreateProblemModalToCreateProblemParams maps parsed form values into a create request.
// Values remain unchanged for service normalization and validation.
func MapCreateProblemModalToCreateProblemParams(modal CreateProblemModal) service_model.CreateProblemParams {
	return service_model.CreateProblemParams{
		Name:       modal.Name,
		URL:        modal.URL,
		ClassCode:  modal.ClassCode,
		Difficulty: modal.Difficulty,
		Topic:      modal.Topic,
		Priority:   modal.Priority,
	}
}

// MapEditProblemModalToUpdateProblemParams maps parsed form values into an update
// request, including the target and destination class code. Values still require
// service normalization and validation before writing.
func MapEditProblemModalToUpdateProblemParams(modal EditProblemModal) service_model.UpdateProblemParams {
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

func mapProblemValuesToModalFields(name, url string, difficulty, topic, priority *string) ProblemModalFields {
	return ProblemModalFields{
		Name: name, URL: url, Difficulty: difficulty, Topic: topic, Priority: priority,
	}
}
