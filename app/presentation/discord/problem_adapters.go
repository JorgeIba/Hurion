package discord

import (
	"strings"

	"github.com/JorgeIba/Hurion/app/presentation"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// MapCreateProblemParamsToCreateProblemModal prefills the creation form from supplied values.
// Missing difficulty and priority use their display defaults; missing topic is blank.
func MapCreateProblemParamsToCreateProblemModal(params service_model.CreateProblemParams) CreateProblemModal {
	return CreateProblemModal{
		ClassCode:          params.ClassCode,
		ProblemModalFields: mapProblemValuesToModalFields(params.Name, params.URL, params.Difficulty, params.Topic, params.Priority),
	}
}

// MapUpdateProblemParamsToEditProblemModal prefills the editor using a class code resolved
// by the caller from params.ClassID. ProblemID is copied only for display.
func MapUpdateProblemParamsToEditProblemModal(params service_model.UpdateProblemParams, classCode string) EditProblemModal {
	return EditProblemModal{
		ProblemID:          params.ProblemID,
		ClassCode:          classCode,
		ProblemModalFields: mapProblemValuesToModalFields(params.Name, params.URL, params.Difficulty, params.Topic, params.Priority),
	}
}

// MapCreateProblemModalToCreateProblemParams converts form text into a pending create request.
// Blank optional text and unassigned metadata markers become nil. The multiword
// priority label maps to its code; other text remains raw for application validation.
func MapCreateProblemModalToCreateProblemParams(modal CreateProblemModal) service_model.CreateProblemParams {
	return service_model.CreateProblemParams{
		Name:       modal.Name,
		URL:        modal.URL,
		ClassCode:  modal.ClassCode,
		Difficulty: mapModalDifficultyToOptionalValue(modal.Difficulty),
		Topic:      mapModalTextToOptionalValue(modal.Topic),
		Priority:   mapModalPriorityToOptionalValue(modal.Priority),
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
		Priority:   mapModalPriorityToOptionalValue(modal.Priority),
	}
}

// mapProblemValuesToModalFields formats canonical metadata codes for display
// while preserving other supplied text for prefill or correction.
func mapProblemValuesToModalFields(name, url string, difficulty, topic, priority *string) ProblemModalFields {
	fields := ProblemModalFields{
		Name: name, URL: url, Difficulty: presentation.DifficultyLabel(difficulty),
		Priority: presentation.PriorityLabel(priority),
	}
	if topic != nil {
		fields.Topic = *topic
	}
	return fields
}

func mapModalDifficultyToOptionalValue(value string) *string {
	if strings.EqualFold(strings.TrimSpace(value), presentation.DifficultyUnspecifiedLabel) {
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

func mapModalPriorityToOptionalValue(value string) *string {
	code := service_model.PriorityNotRequired
	if strings.EqualFold(strings.TrimSpace(value), presentation.PriorityLabel(&code)) {
		return &code
	}

	if strings.EqualFold(strings.TrimSpace(value), presentation.PriorityUnspecifiedLabel) {
		return nil
	}
	return mapModalTextToOptionalValue(value)
}
