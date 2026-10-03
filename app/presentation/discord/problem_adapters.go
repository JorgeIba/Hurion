package discord

import (
	"strings"

	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// MapCreateProblemParamsToCreateProblemModal prefills the creation form from supplied values.
// Missing difficulty uses the existing display default; missing topic is blank.
func MapCreateProblemParamsToCreateProblemModal(params service_model.CreateProblemParams) CreateProblemModal {
	return CreateProblemModal{
		ClassCode:          params.ClassCode,
		ProblemModalFields: mapProblemValuesToModalFields(params.Name, params.URL, params.Difficulty, params.Topic),
	}
}

// MapUpdateProblemParamsToEditProblemModal prefills the editor using a class code resolved
// by the caller from params.ClassID. ProblemID is copied only for display.
func MapUpdateProblemParamsToEditProblemModal(params service_model.UpdateProblemParams, classCode string) EditProblemModal {
	return EditProblemModal{
		ProblemID:          params.ProblemID,
		ClassCode:          classCode,
		ProblemModalFields: mapProblemValuesToModalFields(params.Name, params.URL, params.Difficulty, params.Topic),
	}
}

// MapCreateProblemModalToCreateProblemParams converts form text into a pending create request.
// Blank optional text and unspecified difficulty become nil. Other text stays raw;
// the application must normalize and validate it before writing.
func MapCreateProblemModalToCreateProblemParams(modal CreateProblemModal) service_model.CreateProblemParams {
	return service_model.CreateProblemParams{
		Name:       modal.Name,
		URL:        modal.URL,
		ClassCode:  modal.ClassCode,
		Difficulty: mapModalDifficultyToOptionalValue(modal.Difficulty),
		Topic:      mapModalTextToOptionalValue(modal.Topic),
	}
}

// MapEditProblemModalToUpdateProblemParams converts form text into a pending update request.
// The caller supplies the session's target ID and the class ID resolved from the
// submitted ClassCode. Display metadata is ignored. Values still require application
// normalization and validation before writing, as in MapCreateProblemModalToCreateProblemParams.
func MapEditProblemModalToUpdateProblemParams(modal EditProblemModal, problemID, destinationClassID int64) service_model.UpdateProblemParams {
	return service_model.UpdateProblemParams{
		ProblemID:  problemID,
		Name:       modal.Name,
		URL:        modal.URL,
		ClassID:    destinationClassID,
		Difficulty: mapModalDifficultyToOptionalValue(modal.Difficulty),
		Topic:      mapModalTextToOptionalValue(modal.Topic),
	}
}

// mapProblemValuesToModalFields preserves supplied strings for prefill or correction.
func mapProblemValuesToModalFields(name, url string, difficulty, topic *string) ProblemModalFields {
	fields := ProblemModalFields{
		Name: name, URL: url, Difficulty: service_model.DifficultyDefault,
	}
	if difficulty != nil {
		fields.Difficulty = *difficulty
	}
	if topic != nil {
		fields.Topic = *topic
	}
	return fields
}

func mapModalDifficultyToOptionalValue(value string) *string {
	if strings.EqualFold(strings.TrimSpace(value), service_model.DifficultyDefault) {
		return nil
	}
	return mapModalTextToOptionalValue(value)
}

func mapModalTextToOptionalValue(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
